package parse

import "potano.layercake/fns"


var isNameVerChar, isSlotNameStartChar, isSlotNameMidChar, isRepoNameChar,
	isUseDepChar, IsUseFlagChar fns.CharTypeMap


func init() {
	isNameVerChar = fns.MakeCharTypeMap("a-zA-Z0-9/_+*.-")

	isSlotNameStartChar = fns.MakeCharTypeMap("a-zA-Z0-9_")
	isSlotNameMidChar = fns.MakeCharTypeMap("a-zA-Z0-9+_.-")

	isRepoNameChar = fns.MakeCharTypeMap("a-zA-Z0-9_-")

	isUseDepChar = fns.MakeCharTypeMap("a-zA-Z0-9+_@!?=(),-")
	IsUseFlagChar = fns.MakeCharTypeMap("a-zA-Z0-9+_@-")
}

