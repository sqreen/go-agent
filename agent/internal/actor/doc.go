// Copyright 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

// Package actor manages and associates IP addresses or user IDs to security
// actions such as redirecting, blacklisting or whitelisting.
//
// The stores are grouped according to the way the backend sends them to the
// agent. The goal is to be able to update them separately in order to avoid
// read/write modifications: when a new store is received, it is instantiated
// in the agent and then atomically swapped with the current one. Current
// stores are security actions and the CIDR whitelist
//
// Actor Store
//
// The actor actionStore is a central agent actionStore associating actor IP
// addresses and user identifiers to security actions provided by the backend.
// Since it is used in HTTP request handlers, it is designed to be as efficient
// as possible to avoid slowing down requests. An important design constraint is
// the fact that the sooner a request is handled, the sooner its memory is
// released (goroutines and memory they used). So time-efficiency is considered
// as a better general memory-efficiency here.
//
// Radix Tree
//
// A radix-tree is used to efficiently store security actions by IP addresses
// and networks.
//
// Security Actions
//
// Security actions are stored using structures implementing the Action
// interface. Actions can have a time duration by implementing the Timed
// interface.
//
// Security Action HTTP Handlers
//
// Constructors `NewUserActionHTTPHandler()` and `NewIPActionHTTPHandler()`
// allow to create a `http.Handler` from an action that matched a user or an IP
// address. They allow to apply the expected security response to the request's
// response. The user and IP address are used as properties of events performed
// by handlers.
//
// CIDR Whitelist Store
//
// The CIDR whitelist store is a set of radix trees simply storing CIDRs that
// should be whitelisted. The same action tree is used in order to find back
// the matching source, stored as the action ID. This is required to send the
// expected metric key including the matching CIDR
//
package actor
