package types

import (
	"os"
	"regexp"
)

type FilePerm struct {
	User  Perm
	Group Perm
	Other Perm
}

type Perm struct {
	Read  bool
	Write bool
	Exec  bool
}

var regexpPerm = regexp.MustCompile(`^[d\-]([rwx\-]{3}){3}$`)

func FilePermFrom(mode os.FileMode) FilePerm {
	str := mode.String()
	if !regexpPerm.MatchString(str) {
		panic("unexpected file mode: " + str)
	}

	return FilePerm{
		User:  permForm(str[1:4]),
		Group: permForm(str[4:7]),
		Other: permForm(str[7:10]),
	}
}

func permForm(part string) Perm {
	return Perm{
		Read:  part[0] == 'r',
		Write: part[1] == 'w',
		Exec:  part[2] == 'x',
	}
}

func (p Perm) AnyPermission() bool {
	return p.Read || p.Write || p.Exec
}

func (p Perm) IsFullPermission() bool {
	return p.Read && p.Write && p.Exec
}
