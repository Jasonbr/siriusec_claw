package types

// ChannelId identifies a messaging channel (e.g. "telegram", "dingtalk", "feishu").
type ChannelId = string

// SessionKey is the composite key used to identify a session (e.g. "tg:123:main").
type SessionKey = string
