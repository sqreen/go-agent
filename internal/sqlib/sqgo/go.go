// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqgo

import "strings"

// Unvendor returns the given symbol name without the vendor directory prefix
// if any. For example, given `my-app/vendor/github.com/sqreen/go-agent`,
// the function returns `github.com/sqreen/go-agent`
func Unvendor(symbol string) (unvendored string) {
	vendorDir := "/vendor/"
	i := strings.Index(symbol, vendorDir)
	if i == -1 {
		return symbol
	}
	return symbol[i+len(vendorDir):]
}
