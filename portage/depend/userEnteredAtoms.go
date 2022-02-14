package depend

import "fmt"


type UserEnteredDependencies struct {
	Atoms []*DependAtom
	amap map[string]int
}


func NewUserEnteredDependencies() *UserEnteredDependencies {
	return &UserEnteredDependencies{amap: map[string]int{}}
}


func (ued *UserEnteredDependencies) Add(atomString string) error {
	if _, hit := ued.amap[atomString]; hit {
		return fmt.Errorf("duplicate entry for atom %s", atomString)
	}
	da, err := NewDependencyAtom(atomString)
	if err != nil {
		return err
	}
	ued.Atoms = append(ued.Atoms, da)
	ued.amap[atomString] = len(ued.Atoms) - 1
	return nil
}


func (ued *UserEnteredDependencies) Remove(atomString string) bool {
	i, hit := ued.amap[atomString]
	if !hit {
		return false
	}
	ued.Atoms = append(ued.Atoms[:i], ued.Atoms[i+1:]...)
	delete(ued.amap, atomString)
	for key, val := range ued.amap {
		if val > i {
			ued.amap[key] = i - 1
		}
	}
	return true
}

