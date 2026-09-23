window.MOCK_GRAPH = {
  "nodes": [
    {
      "id": "100000003684369100",
      "role": "consolidator",
      "role_score": 0.82,
      "cluster": 1,
      "priority": 0.95,
      "is_seed": false,
      "depth": 0,
      "evidence": "Тестовые данные: пример роли и связей",
      "in_deg": 3,
      "out_deg": 1,
      "in_kzt": 450000,
      "out_kzt": 200000
    },
    {
      "id": "100000003684369101",
      "role": "transit",
      "role_score": 0.82,
      "cluster": 1,
      "priority": 0.88,
      "is_seed": false,
      "depth": 1,
      "evidence": "Тестовые данные: пример роли и связей",
      "in_deg": 1,
      "out_deg": 1,
      "in_kzt": 100000,
      "out_kzt": 150000
    },
    {
      "id": "100000003684369102",
      "role": "distributor",
      "role_score": 0.82,
      "cluster": 1,
      "priority": 0.81,
      "is_seed": true,
      "depth": 2,
      "evidence": "Тестовые данные: пример роли и связей",
      "in_deg": 0,
      "out_deg": 2,
      "in_kzt": 0,
      "out_kzt": 150000
    },
    {
      "id": "100000003684369103",
      "role": "terminal",
      "role_score": 0.82,
      "cluster": 1,
      "priority": 0.74,
      "is_seed": false,
      "depth": 3,
      "evidence": "Тестовые данные: пример роли и связей",
      "in_deg": 1,
      "out_deg": 0,
      "in_kzt": 200000,
      "out_kzt": 0
    },
    {
      "id": "100000003684369104",
      "role": "coordinator",
      "role_score": 0.82,
      "cluster": 1,
      "priority": 0.67,
      "is_seed": false,
      "depth": 0,
      "evidence": "Тестовые данные: пример роли и связей",
      "in_deg": 0,
      "out_deg": 3,
      "in_kzt": 0,
      "out_kzt": 1150000
    },
    {
      "id": "100000003684369105",
      "role": "peripheral",
      "role_score": 0.82,
      "cluster": 2,
      "priority": 0.6,
      "is_seed": false,
      "depth": 1,
      "evidence": "Тестовые данные: пример роли и связей",
      "in_deg": 2,
      "out_deg": 1,
      "in_kzt": 850000,
      "out_kzt": 350000
    },
    {
      "id": "100000003684369106",
      "role": "consolidator",
      "role_score": 0.82,
      "cluster": 2,
      "priority": 0.53,
      "is_seed": false,
      "depth": 2,
      "evidence": "Тестовые данные: пример роли и связей",
      "in_deg": 2,
      "out_deg": 1,
      "in_kzt": 950000,
      "out_kzt": 400000
    },
    {
      "id": "100000003684369107",
      "role": "transit",
      "role_score": 0.82,
      "cluster": 2,
      "priority": 0.46,
      "is_seed": false,
      "depth": 3,
      "evidence": "Тестовые данные: пример роли и связей",
      "in_deg": 1,
      "out_deg": 1,
      "in_kzt": 400000,
      "out_kzt": 450000
    },
    {
      "id": "100000003684369108",
      "role": "distributor",
      "role_score": 0.82,
      "cluster": 2,
      "priority": 0.39,
      "is_seed": false,
      "depth": 0,
      "evidence": "Тестовые данные: пример роли и связей",
      "in_deg": 1,
      "out_deg": 1,
      "in_kzt": 450000,
      "out_kzt": 500000
    },
    {
      "id": "100000003684369109",
      "role": "terminal",
      "role_score": 0.82,
      "cluster": 2,
      "priority": 0.32,
      "is_seed": false,
      "depth": 1,
      "evidence": "Тестовые данные: пример роли и связей",
      "in_deg": 1,
      "out_deg": 1,
      "in_kzt": 500000,
      "out_kzt": 550000
    }
  ],
  "edges": [
    {
      "source": "100000003684369102",
      "target": "100000003684369100",
      "sum_kzt": 50000,
      "n_tx": 1
    },
    {
      "source": "100000003684369102",
      "target": "100000003684369101",
      "sum_kzt": 100000,
      "n_tx": 2
    },
    {
      "source": "100000003684369101",
      "target": "100000003684369100",
      "sum_kzt": 150000,
      "n_tx": 3
    },
    {
      "source": "100000003684369100",
      "target": "100000003684369103",
      "sum_kzt": 200000,
      "n_tx": 4
    },
    {
      "source": "100000003684369104",
      "target": "100000003684369100",
      "sum_kzt": 250000,
      "n_tx": 5
    },
    {
      "source": "100000003684369104",
      "target": "100000003684369105",
      "sum_kzt": 300000,
      "n_tx": 6
    },
    {
      "source": "100000003684369105",
      "target": "100000003684369106",
      "sum_kzt": 350000,
      "n_tx": 7
    },
    {
      "source": "100000003684369106",
      "target": "100000003684369107",
      "sum_kzt": 400000,
      "n_tx": 8
    },
    {
      "source": "100000003684369107",
      "target": "100000003684369108",
      "sum_kzt": 450000,
      "n_tx": 9
    },
    {
      "source": "100000003684369108",
      "target": "100000003684369109",
      "sum_kzt": 500000,
      "n_tx": 10
    },
    {
      "source": "100000003684369109",
      "target": "100000003684369105",
      "sum_kzt": 550000,
      "n_tx": 11
    },
    {
      "source": "100000003684369104",
      "target": "100000003684369106",
      "sum_kzt": 600000,
      "n_tx": 12
    }
  ],
  "clusters": [
    {
      "id": 1,
      "n_nodes": 5,
      "n_seed": 1,
      "sum_kzt_internal": 1000000,
      "hypothesis": "Демонстрационный кластер"
    },
    {
      "id": 2,
      "n_nodes": 5,
      "n_seed": 0,
      "sum_kzt_internal": 1000000,
      "hypothesis": "Демонстрационный кластер"
    }
  ],
  "top": [
    {
      "rank": 1,
      "gid": "100000003684369100",
      "role": "consolidator",
      "priority": 0.95,
      "why": "Тестовые данные: пример роли и связей"
    },
    {
      "rank": 2,
      "gid": "100000003684369101",
      "role": "transit",
      "priority": 0.88,
      "why": "Тестовые данные: пример роли и связей"
    },
    {
      "rank": 3,
      "gid": "100000003684369102",
      "role": "distributor",
      "priority": 0.81,
      "why": "Тестовые данные: пример роли и связей"
    },
    {
      "rank": 4,
      "gid": "100000003684369103",
      "role": "terminal",
      "priority": 0.74,
      "why": "Тестовые данные: пример роли и связей"
    },
    {
      "rank": 5,
      "gid": "100000003684369104",
      "role": "coordinator",
      "priority": 0.67,
      "why": "Тестовые данные: пример роли и связей"
    },
    {
      "rank": 6,
      "gid": "100000003684369105",
      "role": "peripheral",
      "priority": 0.6,
      "why": "Тестовые данные: пример роли и связей"
    },
    {
      "rank": 7,
      "gid": "100000003684369106",
      "role": "consolidator",
      "priority": 0.53,
      "why": "Тестовые данные: пример роли и связей"
    },
    {
      "rank": 8,
      "gid": "100000003684369107",
      "role": "transit",
      "priority": 0.46,
      "why": "Тестовые данные: пример роли и связей"
    },
    {
      "rank": 9,
      "gid": "100000003684369108",
      "role": "distributor",
      "priority": 0.39,
      "why": "Тестовые данные: пример роли и связей"
    },
    {
      "rank": 10,
      "gid": "100000003684369109",
      "role": "terminal",
      "priority": 0.32,
      "why": "Тестовые данные: пример роли и связей"
    }
  ]
};
