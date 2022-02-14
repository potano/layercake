package parse


type AtomCursor struct {
	Slice []byte
	Pos, Last int
}


func NewAtomCursor(buf []byte) *AtomCursor {
	return &AtomCursor{Slice: buf, Last: len(buf) - 1}
}


func (ac *AtomCursor) Peek() byte {
	if ac.Pos > ac.Last {
		return 0
	}
	return ac.Slice[ac.Pos]
}


func (ac *AtomCursor) Peek1() byte {
	if ac.Pos + 1 > ac.Last {
		return 0
	}
	return ac.Slice[ac.Pos + 1]
}


func (ac *AtomCursor) Peek2() byte {
	if ac.Pos + 2 > ac.Last {
		return 0
	}
	return ac.Slice[ac.Pos + 2]
}


func (ac *AtomCursor) Take() byte {
	ac.Pos++
	return ac.Peek()
}



func (ac *AtomCursor) TakeNameVerChars() int {
	c := ac.Peek()
	for {
		if !isNameVerChar[c] {
			break
		}
		c = ac.Take()
	}
	return ac.Pos
}


func (ac *AtomCursor) TakeSlot() (slot, subslot, slotop string) {
	if ac.Peek() != ':' || ac.Peek1() == ':' {
		return
	}
	var c byte
	for {
		c = ac.Take()
		if !isSlotNameStartChar[c] {
			break
		}
		start := ac.Pos
		for {
			c = ac.Take()
			if !isSlotNameMidChar[c] {
				break
			}
		}
		if len(slot) == 0 {
			slot = string(ac.Slice[start:ac.Pos])
		} else if len(subslot) == 0 {
			subslot = string(ac.Slice[start:ac.Pos])
		} else {
			break
		}
		if c != '/' {
			break
		}
	}
	if c == '*' || c == '=' {
		slotop = string(c)
		ac.Pos++
	}
	return
}


func (ac *AtomCursor) TakeRepo() string {
	if ac.Peek() != ':' || ac.Peek1() != ':' {
		return ""
	}
	ac.Pos += 2
	c := ac.Peek()
	if !isRepoNameChar[c] || c == '-' {
		return ""
	}
	start := ac.Pos
	for {
		c = ac.Take()
		if !isRepoNameChar[c] {
			break
		}
	}
	return string(ac.Slice[start:ac.Pos])
}


func (ac *AtomCursor) TakeUseDependencyString() []byte {
	var c byte
	if ac.Peek() != '[' {
		return nil
	}
	start := ac.Pos
	for {
		c = ac.Take()
		if !isUseDepChar[c] {
			break
		}
	}
	if c != ']' || ac.Pos == start + 1 {
		ac.Pos = start
		return nil
	}
	ac.Pos++
	return ac.Slice[start + 1 : ac.Pos - 1]
}


func (ac *AtomCursor) NextWhitespace() int {
	pos := ac.Pos
	c := ac.Peek()
	for c > ' ' {
		c = ac.Take()
	}
	ac.Pos, pos = pos, ac.Pos
	return pos
}


func (ac *AtomCursor) RemainingToken() string {
	return string(ac.Slice[ac.Pos : ac.NextWhitespace()])
}


func (ac *AtomCursor) RemainingTokenAtPos(pos int) string {
	ac.Pos, pos = pos, ac.Pos
	newStr := ac.RemainingToken()
	ac.Pos = pos
	return newStr
}


func (ac *AtomCursor) SampleAfterPos(pos int) string {
	if pos == 0 || pos > ac.Last {
		return ""
	}
	end := pos + 20
	if end > ac.Last {
		end = ac.Last
	}
	return string(ac.Slice[pos:end])
}

