package maven

import (
	"github.com/vifraa/gopom"
	"murphysec-cli-simple/logger"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

func ReadMavenProject(dir string) ([]*PomFile, error) {
	// load all maven module
	q := []string{filepath.Join(dir, "pom.xml")}
	pomMap := map[string]*PomFile{}
	for len(q) > 0 {
		currentPath := q[0]
		q = q[1:]

		if pomMap[currentPath] != nil {
			continue
		}
		logger.Debug.Println("Reading pom:", currentPath)
		pom, e := readPom(currentPath)
		if e != nil {
			return nil, e
		}
		pomMap[currentPath] = pom
		// add modules to q
		for _, it := range pom.dom.Modules {
			modulePath := filepath.Join(filepath.Dir(currentPath), it, "pom.xml")
			q = append(q, modulePath)
		}
	}
	var rs []*PomFile
	for _, it := range pomMap {
		rs = append(rs, it)
	}
	resolvePomInheritance(rs)
	return rs, nil
}

type PomFile struct {
	// Path is the absolute path of the POM file
	Path   string
	dom    *gopom.Project
	Parent *PomFile
}

func (this *PomFile) Coordination() Coordinate {
	c := Coordinate{
		GroupId:    resolvePomPropertiesValue(this, this.dom.GroupID),
		ArtifactId: resolvePomPropertiesValue(this, this.dom.ArtifactID),
		Version:    resolvePomPropertiesValue(this, this.dom.Version),
	}
	if c.GroupId == "" {
		c.GroupId = resolvePomPropertiesValue(this, this.dom.Parent.GroupID)
	}
	if c.Version == "" {
		c.Version = resolvePomPropertiesValue(this, this.dom.Parent.Version)
	}
	return c
}

func (this *PomFile) _dependencies() map[Coordinate]struct{} {
	m := map[Coordinate]struct{}{}
	if this.Parent != nil {
		for v := range this.Parent._dependencies() {
			m[v] = struct{}{}
		}
	}
	for _, it := range this.dom.Dependencies {
		if !(it.Scope == "" || it.Scope == "compile" || it.Scope == "runtime") {
			mc := Coordinate{
				GroupId:    resolvePomPropertiesValue(this, it.GroupID),
				ArtifactId: resolvePomPropertiesValue(this, it.ArtifactID),
				Version:    resolvePomPropertiesValue(this, it.Version),
			}
			if mc.Version == "" {
				mc.Version = resolvePomDependencyVersion(this, mc.GroupId, mc.ArtifactId)
			}
			m[mc] = struct{}{}
		}
	}
	return m
}
func (this *PomFile) Dependencies() []Coordinate {
	var rs []Coordinate
	for it := range this._dependencies() {
		rs = append(rs, it)
	}
	sort.Slice(rs, func(i, j int) bool {
		if rs[i].GroupId != rs[j].GroupId {
			return rs[i].GroupId < rs[j].GroupId
		}
		if rs[i].ArtifactId != rs[j].ArtifactId {
			return rs[i].ArtifactId < rs[j].ArtifactId
		}
		return rs[i].Version < rs[j].Version
	})
	return rs
}

func readPom(path string) (*PomFile, error) {
	p, e := gopom.Parse(path)
	if e != nil {
		return nil, e
	}
	o := &PomFile{
		Path: path,
		dom:  p,
	}
	return o, nil
}

func resolvePomDependencyVersion(pom *PomFile, groupId, artifactId string) string {
	for pom != nil {
		for _, it := range pom.dom.DependencyManagement.Dependencies {
			if it.GroupID == groupId && it.ArtifactID == artifactId && it.Version != "" {
				return resolvePomPropertiesValue(pom, it.Version)
			}
		}
		pom = pom.Parent
	}
	return ""
}

func resolvePomInheritance(pomFiles []*PomFile) {
	pm := map[string]*PomFile{}
	for _, it := range pomFiles {
		pm[it.Coordination().String()] = it
	}
	for _, it := range pm {
		if it.dom.Parent.ArtifactID == "" {
			continue
		}
		parentCoordination := Coordinate{
			GroupId:    resolvePomPropertiesValue(it, it.dom.Parent.GroupID),
			ArtifactId: resolvePomPropertiesValue(it, it.dom.Parent.ArtifactID),
			Version:    resolvePomPropertiesValue(it, it.dom.Parent.Version),
		}
		it.Parent = pm[parentCoordination.String()]
	}
}

type Coordinate struct {
	GroupId    string
	ArtifactId string
	Version    string
}

func (this Coordinate) String() string {
	s := this.GroupId + ":" + this.ArtifactId
	if this.Version != "" {
		s += ":" + this.Version
	}
	return s
}

var pomInlineParameterPattern = regexp.MustCompile("\\$\\{([^{}]+)\\}")

func resolvePomPropertiesValue(p *PomFile, s string) string {
	rawStr := pomInlineParameterPattern.Split(s, -1)
	if p.dom.Properties.Entries == nil {
		return strings.Join(rawStr, "")
	}
	rs := []string{rawStr[0]}
	matches := pomInlineParameterPattern.FindAllStringSubmatch(s, -1)
	for index, match := range matches {
		rs = append(rs, p.dom.Properties.Entries[match[1]])
		rs = append(rs, rawStr[index+1])
	}
	return strings.Join(rs, "")
}
