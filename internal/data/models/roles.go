package models

// Role — роль узла в финансовой сети (словарь ТЗ).
type Role string

const (
	RoleConsolidator Role = "consolidator" // аккумулирует средства от нескольких участников
	RoleTransit      Role = "transit"      // пропускает средства дальше, не удерживая
	RoleDistributor  Role = "distributor"  // веерное распределение на много получателей
	RoleTerminal     Role = "terminal"     // деньги приходят и остаются
	RoleCoordinator  Role = "coordinator"  // координирующий узел, кандидат в организаторы
	RolePeripheral   Role = "peripheral"   // признаков роли не выявлено
)

var AllRoles = []Role{RoleConsolidator, RoleTransit, RoleDistributor, RoleTerminal, RoleCoordinator, RolePeripheral}

func IsValidRole(role string) bool {
	for _, r := range AllRoles {
		if string(r) == role {
			return true
		}
	}
	return false
}
