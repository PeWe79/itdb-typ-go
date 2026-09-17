package assets

import "net/http"

// assetPermission 拼接资产资源的权限 Key，scope 为 read 或 manage
func assetPermission(resource, scope string) string {
	return "assets." + resource + "." + scope
}

// dictionaryScopeName 合同子类型的读写权限归属合同类型
func dictionaryScopeName(name string) string {
	if name == "contractsubtypes" {
		return "contracttypes"
	}
	return name
}

// dictionaryPermission 拼接资料字典的权限 Key，scope 为 read 或 manage
func dictionaryPermission(name, scope string) string {
	return "dictionaries." + dictionaryScopeName(name) + "." + scope
}

// hasPermission 判断当前请求用户是否持有指定权限
func (a *Router) hasPermission(r *http.Request, permission string) bool {
	if a.currentUserPermissions == nil {
		return false
	}
	for _, item := range a.currentUserPermissions(r) {
		if item == permission {
			return true
		}
	}
	return false
}
