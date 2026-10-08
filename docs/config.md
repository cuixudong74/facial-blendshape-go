# 配置说明

运行时配置是一个 JSON 文件（由 `pkg/config` 读取），形变目标数据是独立的
JSON 文件（由 `pkg/store` 读取、可热替换）。二者分离，使得**替换形变数据
无需重新发版、无需重启**（配合 `DirectoryReloader`）。

## 运行时配置 `config.json`

```json
{
  "vertexCount": 5120,
  "weights": {
    "min": 0.0,
    "max": 1.0,
    "clamp": true,
    "normalize": true,
    "skipInvalid": true
  },
  "targets": [
    {"name": "smile",     "file": "targets/smile.json",     "kind": "base",       "priority": 0},
    {"name": "jaw_open",  "file": "targets/jaw_open.json",  "kind": "additive",   "priority": 0},
    {"name": "lip_corr",  "file": "targets/lip_corr.json",  "kind": "corrective", "priority": 1}
  ],
  "clips": [
    {
      "name": "talk",
      "duration": 2.0,
      "tracks": [
        {"target": "smile", "keys": [
          {"time": 0.0, "value": 0.0},
          {"time": 0.5, "value": 1.0},
          {"time": 1.5, "value": 1.0},
          {"time": 2.0, "value": 0.0}
        ]}
      ]
    }
  ]
}
```

字段：

- `vertexCount`：rig 顶点数，所有目标必须一致，必须为正。
- `weights`：见 README 的 `WeightPolicy`；要求 `min <= max`。
- `targets[]`：`name` 唯一；`kind` 取值 `base|additive|corrective`；
  `priority` 决定同层内施加顺序（小者先）。
- `clips[]`：
  - `duration`（秒）> 0；
  - 每条 `track.target` 对应一个目标名；
  - `keys` 必须时间**严格递增**、有限值；片段内 target 不可重复。

校验失败会返回带字段上下文的错误，拒绝加载。

## 形变目标文件 `*.json`

```json
{
  "name": "smile",
  "kind": "base",
  "priority": 0,
  "vertexCount": 5120,
  "deltas": [ [0.0, 0.12, 0.0], [0.0, 0.10, 0.0] ],
  "checksum": "可选：内容 SHA-256（hex）"
}
```

- `deltas` 长度必须等于 `vertexCount`，每个元素是 `[x,y,z]` delta；
- 任何 NaN/Inf、维度不符、未知 kind 都会被拒绝；
- `checksum` 可省略；若提供则必须与载荷的语义 SHA-256 一致
  （由 `store.ChecksumTarget` 计算：name|kind|vertexCount|所有坐标），
  用于检测磁盘损坏/被篡改；
- 单个文件损坏不会影响同批其它目标，错误以目标名上报；但向 Store
  **发布整批**时，含任何非法目标会整体拒绝、保留旧快照。
