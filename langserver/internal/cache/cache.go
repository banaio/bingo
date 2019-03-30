package cache

import (
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/tools/go/packages"

	"github.com/saibing/bingo/langserver/internal/source"
	"github.com/saibing/bingo/langserver/internal/util"
)

type CacheStyle string

const (
	None     CacheStyle = "none"
	Ondemand CacheStyle = "on-demand"
	Always   CacheStyle = "always"
)

type id2Package map[string]*GlobalPackage
type file2Package map[string]*GlobalPackage
type path2Package map[string]*GlobalPackage

// GlobalCache package cache
type GlobalCache struct {
	mu      sync.RWMutex
	idMap   id2Package
	pathMap path2Package
	fileMap file2Package
}

// debugCache trace package cache
var debugCache = false

// NewCache new a package cache
func NewCache() *GlobalCache {
	return &GlobalCache{
		idMap:   id2Package{},
		pathMap: path2Package{},
		fileMap: file2Package{},
	}
}

func (c *GlobalCache) put(pkg *Package) {
	if c == nil {
		log.Printf("GlobalCache.put: c==nil, pkg.id=%#v\n", pkg.id)
		return
	}

	if debugCache {
		log.Printf("cache %s = %p\n", pkg.id, pkg)
	}

	c.delete(pkg.id)
	p := &GlobalPackage{pkg: pkg, modTime: getPackageModTime(pkg)}
	c.idMap[pkg.id] = p
	c.pathMap[pkg.pkgPath] = p

	for _, file := range pkg.files {
		c.fileMap[util.LowerDriver(file)] = p
	}
}

func (c *GlobalCache) get(id string) *Package {
	if c == nil {
		log.Printf("GlobalCache.get: c==nil, id=%#v\n", id)
		return nil
	}

	pkg := c.idMap[id]

	if debugCache {
		log.Printf("get %s = %p\n", id, pkg)
	}
	return pkg.Package()
}

func (c *GlobalCache) delete(id string) {
	if c == nil {
		log.Printf("GlobalCache.delete: c==nil, id=%#v\n", id)
		return
	}

	if debugCache {
		log.Printf("delete %s %p\n", id, c.idMap[id])
	}

	p := c.idMap[id]
	if p == nil {
		return
	}

	delete(c.idMap, id)
	delete(c.pathMap, p.pkg.pkgPath)

	for _, file := range p.pkg.files {
		delete(c.fileMap, util.LowerDriver(file))
	}
}

func (c *GlobalCache) RLock() {
	if c == nil {
		log.Printf("GlobalCache.RLock: c==nil\n")
		return
	}

	c.mu.RLock()
}

func (c *GlobalCache) RUnlock() {
	if c == nil {
		log.Printf("GlobalCache.RUnlock: c==nil\n")
		return
	}

	c.mu.RUnlock()
}

func (c *GlobalCache) Lock() {
	if c == nil {
		log.Printf("GlobalCache.Lock: c==nil\n")
		return
	}

	c.mu.Lock()
}

func (c *GlobalCache) Unlock() {
	if c == nil {
		log.Printf("GlobalCache.Unlock: c==nil\n")
		return
	}

	c.mu.Unlock()
}

func (c *GlobalCache) clean(idList []string) {
	if c == nil {
		log.Printf("GlobalCache.clean: c==nil, idList=%#v\n", idList)
		return
	}

	if len(idList) == 0 {
		log.Printf("GlobalCache.clean: len(idList)==0, idList=%#v\n", idList)
		return
	}

	c.Lock()
	defer c.Unlock()

	for _, id := range idList {
		c.delete(id)
	}
}

// Get get package by package import path from global cache
func (c *GlobalCache) Get(pkgPath string) *GlobalPackage {
	if c == nil {
		log.Printf("GlobalCache.Get: c==nil, pkgPath=%#v\n", pkgPath)
		return nil
	}

	c.RLock()
	p := c.pathMap[pkgPath]
	c.RUnlock()
	return p
}

func (c *GlobalCache) Put(pkg *Package) {
	if c == nil {
		log.Printf("GlobalCache.Put: c==nil, pkg.id=%#v\n", pkg.id)
		return
	}

	c.Lock()
	defer c.Unlock()
	c.put(pkg)
}

func (c *GlobalCache) Delete(id string) {
	if c == nil {
		log.Printf("GlobalCache.Delete: c==nil, id=%#v\n", id)
		return
	}

	c.Lock()
	defer c.Unlock()
	c.delete(id)
}

// GetByURI get package by filename from global cache
func (c *GlobalCache) GetByURI(filename string) *Package {
	if c == nil {
		log.Printf("GlobalCache.GetByURI: c==nil, filename=%#v\n", filename)
		return nil
	}
	c.RLock()
	p := c.fileMap[util.LowerDriver(filename)]
	c.RUnlock()
	return p.Package()
}

// Walk walk the global package cache
func (c *GlobalCache) Walk(walkFunc source.WalkFunc, ranks []string) error {
	if c == nil {
		log.Printf("GlobalCache.Walk: c==nil, ranks=%#v\n", ranks)
		return nil
	}

	c.RLock()
	defer c.RUnlock()

	var idList []string
	for id := range c.idMap {
		idList = append(idList, id)
	}

	getRank := func(id string) int {
		var i int
		for i = 0; i < len(ranks); i++ {
			if strings.HasPrefix(id, ranks[i]) {
				return i
			}
		}

		if strings.Contains(id, ".") {
			return i
		}

		return i + 1
	}

	sort.Slice(idList, func(i, j int) bool {
		r1 := getRank(idList[i])
		r2 := getRank(idList[j])
		if r1 < r2 {
			return true
		}

		if r1 == r2 {
			return idList[i] <= idList[j]
		}

		return false
	})

	return c.walk(idList, walkFunc)
}

func (c *GlobalCache) walk(idList []string, walkFunc source.WalkFunc) error {
	for _, id := range idList {
		pkg := c.get(id)
		if err := walkFunc(pkg); err != nil {
			return err
		}
	}

	return nil
}

func (c *GlobalCache) Add(pkg *packages.Package) {
	if c == nil {
		log.Printf("GlobalCache.Add: c==nil, pkg.id=%#v\n", pkg.ID)
		return
	}

	c.Lock()
	defer c.Unlock()

	c.recusiveAdd(pkg, nil)
}

func (c *GlobalCache) recusiveAdd(pkg *packages.Package, parent *Package) {
	if p, _ := c.idMap[pkg.ID]; p != nil {
		if parent != nil {
			parent.imports[pkg.PkgPath] = p.pkg
		}
		return
	}

	p := create(pkg)

	for _, ip := range pkg.Imports {
		c.recusiveAdd(ip, p)
	}

	c.put(p)

	if parent != nil {
		parent.imports[p.pkgPath] = p
	}
}

func create(pkg *packages.Package) *Package {
	return &Package{
		name:      pkg.Name,
		id:        pkg.ID,
		pkgPath:   pkg.PkgPath,
		files:     pkg.CompiledGoFiles,
		syntax:    pkg.Syntax,
		errors:    pkg.Errors,
		types:     pkg.Types,
		typesInfo: pkg.TypesInfo,
		fset:      pkg.Fset,
		imports:   make(map[string]*Package),
	}
}

func getPackageModTime(pkg *Package) time.Time {
	if pkg == nil || len(pkg.files) == 0 {
		return time.Time{}
	}

	dir := pkg.files[0]
	fi, err := os.Stat(dir)
	if err != nil {
		return time.Time{}
	}

	return fi.ModTime()
}
