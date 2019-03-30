package cache

import "time"

type GlobalPackage struct {
	pkg     *Package
	modTime time.Time
}

func (p *GlobalPackage) Package() *Package {
	if p == nil {
		return nil
	}
	return p.pkg
}

func (p *GlobalPackage) ModTime() time.Time {
	if p == nil {
		return time.Time{}
	}
	return p.modTime
}
