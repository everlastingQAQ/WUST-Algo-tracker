package task

func ActiveRefreshConflictCondition(platform string) (string, []interface{}) {
	if platform == "" {
		return "current_platform = '' OR total_platforms <> 1", nil
	}
	return "current_platform = ? OR current_platform = '' OR total_platforms <> 1", []interface{}{platform}
}
