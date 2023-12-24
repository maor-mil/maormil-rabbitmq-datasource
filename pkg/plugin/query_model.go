package plugin

type QueryModel struct {
	AreMessagesBase64Encrypted bool              `json:"areMessagesBase64Encrypted"`
	JsonQueryModels            []*JsonQueryModel `json:"jsonQueryModels"`
}

type JsonQueryModel struct {
	JsonKeyPath string `json:"jsonKeyPath"`
	RegexValue  string `json:"regexValue"`
}
