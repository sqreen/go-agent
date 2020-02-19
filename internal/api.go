// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package internal

//type AgentFace interface {
//Config() ConfigFace
//FindActionByUserID(userID event.UserIdentifierMap) (action actor.Action, exists bool)
//FindActionByIP(ip net.IP) (action actor.Action, exists bool, err error)
//Logger() *plog.Logger
//CloseRequestContext(RequestContextFace) error
//}

//type ConfigFace interface {
//	PrioritizedIPHeader() string
//	PrioritizedIPHeaderFormat() string
//}

//type RequestContextFace interface {
//	EventRecord() event.Recorded
//	Request() HTTPRequestFace
//	Response() ResponseFace
//}

//func Agent() AgentFace {
//	agent := agentInstance.Get()
//	if agent == nil || agent.IsDisabled() {
//		return nil
//	}
//	return (*publicAgent)(agent)
//}

//type publicAgent AgentType

//func (a *publicAgent) unwrap() *AgentType { return (*AgentType)(a) }

//func (a *publicAgent) FindActionByUserID(userID event.UserIdentifierMap) (action actor.Action, exists bool) {
//	return a.unwrap().FindActionByUserID(userID)
//}
//
//func (a *publicAgent) FindActionByIP(ip net.IP) (action actor.Action, exists bool, err error) {
//	return a.unwrap().FindActionByIP(ip)
//}

//func (a *publicAgent) Config() ConfigFace {
//	return (*publicConfig)(a.unwrap().config)
//}

//func (a *publicAgent) Logger() *plog.Logger {
//	return a.unwrap().logger
//}

//type publicConfig config.Config
//
//func (c *publicConfig) unwrap() *config.Config { return (*config.Config)(c) }
//
//func (c *publicConfig) PrioritizedIPHeader() string {
//	return c.unwrap().HTTPClientIPHeader()
//}
//
//func (c *publicConfig) PrioritizedIPHeaderFormat() string {
//	return c.unwrap().HTTPClientIPHeaderFormat()
//}
