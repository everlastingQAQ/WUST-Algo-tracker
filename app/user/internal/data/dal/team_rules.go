package dal

func CanManageTeam(operatorId, ownerId int64) bool {
	return operatorId > 0 && ownerId > 0 && operatorId == ownerId
}
