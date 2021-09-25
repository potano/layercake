package fs

var ReadOK, WriteOK func (string, ...interface{}) bool

func init() {
	ReadOK = func (m string, p...interface{}) bool { return true }
	WriteOK = ReadOK
}

