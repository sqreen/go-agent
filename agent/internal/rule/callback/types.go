// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package callback

type Context interface {
	// Get the rule configuration.
	Config() interface{}
	// Add a new metrics value for the given key to the default metrics store
	// given by the rule.
	AddMetricsValue(key interface{}, value uint64)
}
