// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package testmock

//
//import (
//	"net/http"
//	"time"
//
//	"github.com/stretchr/testify/mock"
//)
//
//type RequestRecordMockup struct {
//	mock.Mock
//}
//
//func (rr *RequestRecordMockup) Request() *http.Request {
//	return rr.Called().Get(0).(*http.Request)
//}
//
//func (rr *RequestRecordMockup) SetRequest(r *http.Request) {
//	rr.Called(r)
//}
//
//func (rr *RequestRecordMockup) NewCustomEvent(event string) types.CustomEvent {
//	rr.Called(event)
//	return rr
//}
//
//func (rr *RequestRecordMockup) ExpectTrackEvent(event string) *mock.Call {
//	return rr.On("AddCustomEvent", event)
//}
//
//func (rr *RequestRecordMockup) Whitelisted() bool {
//	return rr.Called().Bool(0)
//}
//
//func (rr *RequestRecordMockup) ExpectWhitelisted() *mock.Call {
//	return rr.On("Whitelisted")
//}
//
//func (rr *RequestRecordMockup) Close() {
//	rr.Called()
//}
//
//func (rr *RequestRecordMockup) ExpectClose() *mock.Call {
//	return rr.On("Close")
//}
//
//func (rr *RequestRecordMockup) NewUserAuth(id map[string]string, success bool) {
//	rr.Called(id, success)
//}
//
//func (rr *RequestRecordMockup) ExpectTrackAuth(id map[string]string, success bool) *mock.Call {
//	return rr.On("NewUserAuth", id, success)
//}
//
//func (rr *RequestRecordMockup) NewUserSignup(id map[string]string) {
//	rr.Called(id)
//}
//
//func (rr *RequestRecordMockup) ExpectTrackSignup(id map[string]string) *mock.Call {
//	return rr.On("AddUserSignup", id)
//}
//
//func (rr *RequestRecordMockup) Identify(id map[string]string) {
//	rr.Called(id)
//}
//
//func (rr *RequestRecordMockup) SecurityResponse() http.Handler {
//	ret := rr.Called().Get(0)
//	if ret == nil {
//		return nil
//	}
//	return ret.(http.Handler)
//}
//
//func (rr *RequestRecordMockup) ExpectSecurityResponse() *mock.Call {
//	return rr.On("SecurityResponse")
//}
//
//func (rr *RequestRecordMockup) UserSecurityResponse() http.Handler {
//	ret := rr.Called().Get(0)
//	if ret == nil {
//		return nil
//	}
//	return ret.(http.Handler)
//}
//
//func (rr *RequestRecordMockup) ExpectUserSecurityResponse() *mock.Call {
//	return rr.On("UserSecurityResponse")
//}
//
//func (rr *RequestRecordMockup) ExpectIdentify(id map[string]string) *mock.Call {
//	return rr.On("Identify", id)
//}
//
//func (rr *RequestRecordMockup) WithTimestamp(t time.Time) {
//	rr.Called(t)
//}
//
//func (rr *RequestRecordMockup) ExpectWithTimestamp(t time.Time) *mock.Call {
//	return rr.On("WithTimestamp", t)
//}
//
//func (rr *RequestRecordMockup) WithProperties(props types.EventProperties) {
//	rr.Called(props)
//}
//
//func (rr *RequestRecordMockup) ExpectWithProperties(props types.EventProperties) *mock.Call {
//	return rr.On("WithProperties", props)
//}
//
//func (rr *RequestRecordMockup) WithUserIdentifiers(id map[string]string) {
//	rr.Called(id)
//}
//
//func (rr *RequestRecordMockup) ExpectWithUserIdentifiers(id map[string]string) *mock.Call {
//	return rr.On("WithUserIdentifiers", id)
//}
