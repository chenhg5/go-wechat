package main

// 从数据库初始化所有应用账号对象，存进内存中

var Account = make(map[int]map[string]string)

func InitAccount() {

	account, _ := Query("select acid,app_id,app_secret from wx_official_account where state = 1")

	for i := 0; i < len(account); i++ {
		Account[int(account[i]["acid"].(int64))] = map[string]string{
			"appId": account[i]["app_id"].(string),
			"appSecret": account[i]["app_secret"].(string),
		}
	}
}

func GetAccountInfo(accountid int) map[string]string {
	return Account[accountid]
}