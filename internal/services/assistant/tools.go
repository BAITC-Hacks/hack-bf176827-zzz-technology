package assistant

import (
	"encoding/json"
	"fmt"

	"hackaton/pkg/llm"
)

// toolDefinitions — инструменты для function calling; все читают результат анализа в памяти.
func toolDefinitions() []llm.Tool {
	object := func(properties map[string]any, required ...string) map[string]any {
		schema := map[string]any{"type": "object", "properties": properties, "additionalProperties": false}
		if len(required) > 0 {
			schema["required"] = required
		}
		return schema
	}
	str := map[string]any{"type": "string"}
	integer := map[string]any{"type": "integer"}
	strList := map[string]any{"type": "array", "items": str}
	return []llm.Tool{
		{Name: "get_node", Description: "Карточка узла: роль, скоры, evidence, метрики, топ входящих и исходящих контрагентов с суммами.",
			Parameters: object(map[string]any{"gid": str}, "gid")},
		{Name: "find_nodes", Description: "Поиск узлов по фильтрам, отсортировано по приоритету. role: consolidator|transit|distributor|terminal|coordinator|peripheral.",
			Parameters: object(map[string]any{"role": str, "cluster_id": integer, "is_seed": map[string]any{"type": "boolean"}, "min_priority": map[string]any{"type": "number"}, "limit": integer})},
		{Name: "who_receives_from", Description: "Кто получает деньги от заданных узлов вниз по цепочке (≤ depth шагов): общие получатели, прямые суммы от каждого источника, роли. Используй для «кто собирает деньги с этих N».",
			Parameters: object(map[string]any{"gids": strList, "depth": integer}, "gids")},
		{Name: "who_pays_to", Description: "Кто платит узлу вверх по цепочке (≤ depth шагов): плательщики, суммы, роли, сколько среди них seed.",
			Parameters: object(map[string]any{"gid": str, "depth": integer}, "gid")},
		{Name: "path", Description: "Кратчайший маршрут денег от src к dst (≤ max_len шагов) с суммами по рёбрам.",
			Parameters: object(map[string]any{"src": str, "dst": str, "max_len": integer}, "src", "dst")},
		{Name: "cluster_info", Description: "Кластер: размер, seed, обороты, состав ролей, топ-узлы, гипотеза.",
			Parameters: object(map[string]any{"cluster_id": integer}, "cluster_id")},
		{Name: "top", Description: "Топ узлов по приоритету для проверки (опционально по роли).",
			Parameters: object(map[string]any{"n": integer, "role": str})},
		{Name: "remove_nodes", Description: "Что будет с сетью, если изъять (заблокировать) узлы: сколько компонент, какая доля оборота отвалится, кто потеряет всех плательщиков.",
			Parameters: object(map[string]any{"gids": strList}, "gids")},
	}
}

// executeTool — диспетчер вызовов модели; результат сериализуется в JSON.
func (s *service) executeTool(name, argsJSON string) (string, error) {
	var args toolArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("bad tool args: %w", err)
	}
	var payload any
	var err error
	switch name {
	case "get_node":
		payload, err = s.queryNode(args.GID)
	case "find_nodes":
		payload = s.queryFindNodes(args)
	case "who_receives_from":
		payload, err = s.queryReceiversOf(args.GIDs, args.Depth)
	case "who_pays_to":
		payload, err = s.queryPayersOf(args.GID, args.Depth)
	case "path":
		payload, err = s.queryPath(args.Src, args.Dst, args.MaxLen)
	case "cluster_info":
		payload, err = s.queryCluster(args.ClusterID)
	case "top":
		payload = s.queryTop(args.N, args.Role)
	case "remove_nodes":
		payload, err = s.queryRemoveNodes(args.GIDs)
	default:
		return "", fmt.Errorf("%w: %s", ErrUnknownTool, name)
	}
	if err != nil {
		return "", err
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}
