# HR 个人生命周期事件

先读事件产品入口 [SKILL.md](../SKILL.md) 的命令规则、生命周期和失败处理。本参考覆盖员工转正、调岗、入职与离职四个 HR 个人事件。

<!-- dws-intent: event.listen.hr -->员工转正、调岗、入职或离职生命周期变化必须使用 `dws event consume` 长连接，不轮询员工档案或审批记录模拟事件。

## Event catalog

| 事件码 | 订阅规则 | 接收语义 | 必填参数 |
|---|---|---|---|
| `user_hrm_regular_lifecycle_changed` | `all` | 当前用户的员工转正生命周期发生变化 | 无 |
| `user_hrm_transfer_lifecycle_changed` | `all` | 当前用户的员工调岗生命周期发生变化 | 无 |
| `user_hrm_entry_lifecycle_changed` | `all` | 当前用户的员工入职生命周期发生变化 | 无 |
| `user_hrm_termination_lifecycle_changed` | `all` | 当前用户的员工离职生命周期发生变化 | 无 |

CLI 使用当前 OAuth 用户身份，为每个事件发送 `ruleType=all` 的独立订阅请求。HR Provider 不支持过滤规则，因此请求省略 `filterRule`。不要添加 `--user`、`--open-dingtalk-id`、`--group`、`--query` 或 `--filter-json`。

## Commands

```bash
dws event list --category hr
dws event schema user_hrm_regular_lifecycle_changed --flatten
dws event schema user_hrm_transfer_lifecycle_changed --flatten
dws event schema user_hrm_entry_lifecycle_changed --flatten
dws event schema user_hrm_termination_lifecycle_changed --flatten

dws event consume user_hrm_regular_lifecycle_changed --flatten -f ndjson
dws event consume user_hrm_transfer_lifecycle_changed --flatten -f ndjson
dws event consume user_hrm_entry_lifecycle_changed --flatten -f ndjson
dws event consume user_hrm_termination_lifecycle_changed --flatten -f ndjson
```

四个事件可以共享一个消费生命周期：

```bash
dws event consume \
  user_hrm_regular_lifecycle_changed \
  user_hrm_transfer_lifecycle_changed \
  user_hrm_entry_lifecycle_changed \
  user_hrm_termination_lifecycle_changed \
  --flatten \
  -f ndjson
```

## Output contract

HR 服务尚未发布经过评审的 payload 字段契约，因此 `--flatten` 只稳定承诺公共字段和开放 `payload`：

```json
{
  "type": "user_hrm_regular_lifecycle_changed",
  "event_id": "...",
  "timestamp": 0,
  "subscribe_id": "...",
  "payload": {}
}
```

- `type` 固定为当前 event key；`event_id` 可用于 transport 去重；`subscribe_id` 标识独立订阅。
- `payload` 保留服务端实际业务字段，不应猜测员工 ID、岗位、部门、状态或操作者字段。
- payload 缺失、为空或无法解析时，stderr 输出 warning，stdout 回退为原始 transport envelope，避免静默丢事件。
- 不传 `--flatten` 时保持兼容 transport envelope，完整业务对象位于 `.data | fromjson`。

## Backend boundary

DWS CLI 仍通过统一控制面创建和取消个人订阅。DWS 后端识别这四个 eventKey 后调用 HR 预发 HSF Provider：

- Service：`com.dingtalk.hrmp.process.service.dws.HrmLifecycleDwsSubscriptionService`
- Version：`1.0.0`
- Group：`HSF`
- 注册：`registerSubscription(UserSubscribeRequest)`
- 取消：`unregisterSubscription(UserUnsubscribeRequest)`

中心预发的动态路由位于 unit `pre`、group/app `lippi-open-callback`：

- `dws.event.source.conf`：`category=HRM`，四个 eventKey 均声明 `eventSource=hrm`。
- `dws.subscription.provider.conf`：`providers.hrm` 指向上述 HSF Service，`serviceVersion=1.0.0`、`group=HSF`、`timeoutMs=3000`、`enabled=true`。

CLI 不直连 HSF。订阅或取消失败时保留服务端错误、request/trace 信息交给 DWS 与 HR 后端联调；不要换 eventKey、subscribe_id 或反复创建来绕过重试保护。待 HR 补充真实推送样例和字段契约后，再把开放 `payload` 升级为强类型顶层字段。
