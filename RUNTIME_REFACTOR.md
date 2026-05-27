# grpc-runtime 高性能重构方案

## 方向结论

动态生成没有问题，问题是旧 `protoc-gen-grpc-gateway` 模板生成得太低层

新的边界应该是：

```text
protoc-gen-grpc-gateway  继续动态生成，但只生成 RouteDesc、BindingDesc 和 typed invoker
grpc-runtime             统一承载路由、绑定、校验、metadata、response pipeline
go-rpc-gateway           只负责把 Argus、日志、链路、指标等上层能力注入 runtime
apex-share-proto         只作为旧行为回归样本，不把旧 .pb.gw.go 形态固化成接口
```

旧生成代码里的 `runtime.NewPattern(...)`、`utilities.DoubleArray{...}`、`forward_Xxx` 仍要保留兼容，但它们不应该继续成为新生成器的 public contract

## 当前实现事实

这一版 runtime 已经不是“从零开始重构”，而是进入第二阶段：热路径已经拆出来，模块下沉已经开始，剩下的关键是把 generator 和 Argus 接上

| 区域 | 当前状态 | 说明 |
|------|----------|------|
| `routing` | 已落地 | `Table[T]`、`StaticIndex[T]`、`Params`、`PathBuffer`、template 编译、trie、fallback、对象池和测试已存在 |
| `ServeMux` | 已接入新路由表 | `ServeHTTP` 通过 `routes.Match` 查找路由；动态旧 Pattern 先用 template/trie 预过滤，再用旧 matcher 校准语义 |
| `validation` | 已落地 | 子包定义 `Validator`/`ErrorFormatter`；根包导出 `WithRequestValidator`、`WithValidationErrorFormatter`、`WithValidationSkipper`、`ValidateRequest` |
| `scalar` | 已下沉 | 根包 `scalar.go` 作为兼容 facade，实际转换逻辑在 `scalar/` |
| `codec` | 已下沉一部分 | 子包已有 marshaler/registry/jsonpb/proto/httpbody；根包 `codec.go` 仍承担兼容和 mux registry facade |
| `metadata` | 已下沉一部分 | 子包已有 annotate/header/timeout/stream；根包 `context.go` 保留 context key、兼容函数和 mux 入口 |
| `response` | 已下沉一部分 | 子包已有 status/error/forward 基础能力；根包 `response.go` 仍保留 mux-aware error/forward 行为 |
| `binding` | 原型已开始 | 子包已有 query/fieldmask 初版；根包 `binding.go` 仍是旧生成代码实际依赖的完整兼容实现 |
| generator | 注册形态已切换 | 新模板通过 `runtime.RegisterRoutes` 注册 `[]runtime.RouteDesc`，不再生成 `NewPattern`、`DoubleArray`、`forward_Xxx`、`filter_Xxx`、`mux.Handle` |
| go-rpc-gateway Argus | 未接入 runtime | 旧 middleware 二次读 body 路径仍未废弃 |

已验证：

```bash
cd grpc-runtime
go test ./...
```

## 已收敛的实现优化

### 路由表

当前 `routing.Table` 使用三层路径：

```text
static exact index -> template/trie positive path -> legacy Pattern fallback
```

实现原则：

- 静态路由只进 `StaticIndex`，不再为了 `MatchOther` 重复塞进 fallback
- `Table` 单独维护 method 集合，用于 404/405 和 POST path-length fallback 判断
- 简单动态模板走 trie；复杂模板如 `{name=projects/*}` 不强行编译，交给旧 Pattern fallback，避免 trie 粗匹配扩大语义
- 带 `:verb` 的模板在 trie 叶子节点校验 verb，避免 `/v1/users/123` 误命中 `/v1/users/{id}:get`
- 旧 Pattern 路由即使命中 trie，也会再调用旧 matcher 校准 path param、verb、unescape 语义

### 参数容器

`routing.Params` 已经采用 slice 存储，并懒加载：

- `Get(name)` 首次构建 `index`
- `Map()` 首次构建兼容旧 `HandlerFunc` 的 map
- `Reset()` 清理残留值，配合 pool 复用

注意：池化 API 已存在，但当前 `ServeMux` 的旧 `HandlerFunc(map[string]string)` 仍需要 `Map()`，所以静态/动态端到端 0 alloc 需要等新 `RouteHandler(*Params)` 和新生成器一起切换

### 模块下沉

现在根包不是“纯 facade”，而是兼容 facade + 旧生成代码入口不要继续在根包新增大块实现；新能力优先放到子包，再由根包做薄封装

## 还存在的重复

### `binding.go` 与 `binding/`

这是当前最大的重复

`binding.go` 仍是旧生成代码依赖的完整 query/fieldmask 实现，不能直接删除`binding/` 是新 pipeline 的雏形，但还没有覆盖旧 `PopulateQueryParameters(msg, values, *utilities.DoubleArray)` 的完整语义，也没有接入 generator

处理方向：

1. 先在 `binding/` 定义生成器要用的 `Desc`、`BodyBinding`、`FieldBinding`、`BuildRequest`
2. `BuildRequest` 内部复用当前成熟的根包 query/fieldmask 逻辑，先保证行为不变
3. generator 切到 `RouteDesc` 后，再把根包 `binding.go` 缩成 alias/wrapper
4. 最后废弃 `utilities.DoubleArray` 作为生成代码接口

### `response.go` 与 `response/`

`response/` 目前只有低层 forward/error/status，根包 `response.go` 仍包含 mux 配置、metadata trailer、rewriter、stream chunk error 等完整行为

处理方向：

1. `response/` 接收一个小的 `ForwardConfig`，承载 mux 当前依赖的 matcher、rewriter、options、writeContentLength
2. 根包 `ForwardResponseMessage` 和 `ForwardResponseStream` 改成薄 wrapper
3. 旧函数签名保留，避免破坏旧 `.pb.gw.go`

### `metadata/context.go` 与根包 `context.go`

子包已经负责 header/timeout/metadata 注入，根包仍维护 `RPCMethod`、`HTTPPattern` 等兼容 context key

处理方向：

- `metadata` 继续只管 HTTP/gRPC metadata 互转
- `RPCMethod`、`HTTPPathPattern`、`HTTPPattern` 暂时留根包，因为旧生成代码和外部用户可能直接依赖
- 等 RouteDesc 稳定后，再考虑 request-scoped state，减少 `context.WithValue` 热路径分配

## 下一步开发顺序

### 1. 补 runtime descriptor facade

先在根包补齐新生成器要依赖的稳定入口：

```go
type RouteDesc struct {
    Method      string
    Template    string
    Operation   string
    Request     func() proto.Message
    Body        binding.BodyBinding
    QueryFilter binding.QueryFilter
    Invoker     RouteInvoker
}

func NoBody() binding.BodyBinding
func QueryFilter(fields ...string) binding.QueryFilter
func RegisterRoutes(ctx context.Context, mux *ServeMux, routes []RouteDesc, invoker any) error
```

这一层只做 facade，不把 generator 绑到子包细节上

### 2. 做 `binding.BuildRequest`

请求构建顺序固定为：

```text
new request message
decode body
apply path params
apply query params
apply field mask
ValidateRequest(ctx, mux, r, msg)
```

第一版可以内部调用旧根包逻辑，先减少生成器重复；后续再逐步把旧 `binding.go` 内容移入子包

### 3. 改 generator 输出

目标输出：

```go
var userRoutes = []runtime.RouteDesc{
    {
        Method:      http.MethodGet,
        Template:    "/v1/users/{user_id}",
        Operation:   "/apex.api.UserService/UserGet",
        Request:     func() proto.Message { return new(UserGetRequest) },
        Body:        runtime.NoBody(),
        QueryFilter: runtime.QueryFilter("user_id"),
        Invoker:     invoke_UserService_UserGet_0,
    },
}
```

生成代码只保留必须强类型的 invoker：

```go
func invoke_UserService_UserGet_0(ctx context.Context, req proto.Message, target any) (proto.Message, runtime.ServerMetadata, error) {
    in := req.(*UserGetRequest)
    c := target.(UserServiceClient)
    var md runtime.ServerMetadata
    out, err := c.UserGet(ctx, in, grpc.Header(&md.HeaderMD), grpc.Trailer(&md.TrailerMD))
    return out, md, err
}
```

删除新模板里的：

```text
runtime.NewPattern
utilities.DoubleArray
forward_Xxx
重复 request/local request 构建流程
```

当前落地状态：

- 注册入口已经一刀切到 `runtime.RegisterRoutes(ctx, mux, []runtime.RouteDesc{...})`
- `pattern_Xxx`、`filter_Xxx`、`forward_Xxx`、`mux.Handle(...)` 已从新模板删除
- request/local request 函数已接入 `runtime.ValidateRequest(ctx, mux, req, &protoReq)`
- `runtime.QueryFilter(...)`、`runtime.IOReaderFactory(...)` 已作为根包 facade 给生成代码使用
- typed invoker 所需的 `RouteDesc.Request`、`RouteDesc.Invoker`、`BuildRequest` 已在 runtime 就位，后续可以继续压缩 request/local request 函数，只保留真正需要强类型的 gRPC 调用

### 4. 接 go-rpc-gateway Argus

runtime 只认最小接口：

```go
type Validator interface {
    Struct(any) error
}
```

go-rpc-gateway 创建 mux 时注入：

```go
runtime.WithRequestValidator(argusValidator)
runtime.WithValidationErrorFormatter(formatArgusError)
```

旧 HTTP middleware 校验路径只保留 deprecated compatibility，不再要求用户 `RegisterGatewayMessageType`，也不再二次读 body

### 5. 清理根包重复

顺序必须在 generator 切换之后：

1. `binding.go` 缩成 facade
2. `response.go` 缩成 facade
3. `context.go` 只保留兼容 context key 和 facade
4. `options.go` 拆出 `ServeMuxOption`
5. `compat.go` 集中放 `NewPattern`、`MustPattern`、`DoubleArray` 相关 deprecated 说明

## 注释与模块风格

保持 go-argus / go-toolbox 风格：

- 每个 `.go` 文件有文件头块注释
- 每个包有 `doc.go` 和 `// Package xxx ...` 包注释
- 导出类型/函数使用中文注释，紧贴声明
- 内部注释只解释原因，不复述代码
- 注释末尾不加句号
- 测试统一用 `github.com/stretchr/testify/assert`，表驱动优先

实现风格：

- 热路径优先直白代码，不为了短写使用 `mathx.IF` 这类可读性弱的辅助
- `safe.SafeAccess` 这类 reflect 工具只能放初始化/配置路径，不进入路由、binding、response 热路径
- `syncx.Pool[T]` 可以继续用于 `Params`、`PathBuffer`、body buffer，但必须明确对象生命周期，避免 handler 后仍被使用

## 性能验收

至少保留这些 benchmark：

```text
BenchmarkServeMux_StaticRoute_100
BenchmarkServeMux_StaticRoute_10000
BenchmarkServeMux_DynamicRoute_100
BenchmarkServeMux_DynamicRoute_10000
BenchmarkServeMux_PathParams
BenchmarkServeMux_QueryBinding
BenchmarkServeMux_ValidatedRequest
BenchmarkRouteTable_Match
BenchmarkRouteTable_Register
```

验收目标：

- 静态路由接近 0 alloc/op
- 新 RouteDesc 动态路由不随同 method 下 route 总数线性增长
- 旧 Pattern 动态路由至少能通过 trie 预过滤减少正向匹配成本
- Argus 校验不重复读取 body，不重复 decode
- 新 `.pb.gw.go` 不再出现 `runtime.NewPattern`、`utilities.DoubleArray`、`forward_Xxx`

## 代码风格落地进度

### 注释规范

| 文件 | 文件头注释 | 中文注释 | 去句号 | 状态 |
|------|-----------|---------|--------|------|
| `doc.go` | ✅ | ✅ | ✅ | 完成 |
| `mux.go` | ✅ | ✅ | ✅ | 完成 |
| `pattern.go` | ✅ | ✅ | ✅ | 完成 |
| `export.go` | ✅ | ✅ | ✅ | 完成 |
| `context.go` | ✅ | ✅ | ✅ | 完成 |
| `binding.go` | ✅ | ✅ | ✅ | 完成 |
| `response.go` | ✅ | ✅ | ✅ | 完成 |
| `codec.go` | ✅ | ✅ | ✅ | 完成 |
| `scalar.go` | ✅ | ✅ | ✅ | 完成 |
| `validation.go` | ✅ | ✅ | ✅ | 完成 |

### 测试规范

| 文件 | assert包 | 紧凑表驱动 | 残留t.Errorf | 状态 |
|------|---------|-----------|-------------|------|
| `pattern_test.go` | ✅ | ✅ | 无 | 完成 |
| `context_test.go` | ✅ | ✅ | 无 | 完成 |
| `mux_test.go` | ✅ | ✅ | 无 | 完成 |
| `handler_test.go` | ✅ | ✅ | 0 | 完成 |
| `fieldmask_test.go` | ✅ | ✅ | 无 | 完成 |
| `query_test.go` | ✅ | ✅ | 无 | 完成 |
| `marshal_jsonpb_test.go` | ✅ | ✅ | 无 | 完成 |
| `marshal_json_test.go` | ✅ | ✅ | 0 | 完成 |
| `marshal_proto_test.go` | ✅ | ✅ | 无 | 完成 |
| `marshaler_registry_test.go` | ✅ | ✅ | 无 | 完成 |
| `errors_test.go` | ✅ | ✅ | 无 | 完成 |
| `convert_test.go` | ✅ | ✅ | 无 | 完成 |
| `marshal_httpbodyproto_test.go` | ✅ | ✅ | 无 | 完成 |
| `mux_internal_test.go` | ✅ | ✅ | 无 | 完成 |
| `query_fuzz_test.go` | N/A | N/A | N/A | 无需改 |
| **子包** | | | | |
| `httprule/parse_test.go` | ✅ | ✅ | 0 | 完成 |
| `httprule/types_test.go` | ✅ | ✅ | 0 | 完成 |
| `httprule/compile_test.go` | ✅ | ✅ | 0 | 完成 |
| `utilities/trie_test.go` | ✅ | ✅ | 0 | 完成 |
| `utilities/string_array_flag_test.go` | ✅ | ✅ | 0 | 完成 |

### 根目录文件清单

| 文件 | 职责 | 行数 | 下沉状态 |
|------|------|------|---------|
| `doc.go` | 包文档 | 15 | N/A |
| `mux.go` | 核心路由分发器 | 521 | 已接入routing子包 |
| `pattern.go` | 路径模式匹配(opcode VM) | 408 | 独立，待compat化 |
| `export.go` | 日志接口导出 | 44 | 已委托logging |
| `context.go` | 上下文注解兼容层 | 158 | 已委托metadata |
| `binding.go` | 请求参数绑定(完整实现) | 492 | **最大重复**，待缩facade |
| `response.go` | 响应处理(完整实现) | 383 | 待缩facade |
| `codec.go` | 编解码兼容层+注册表 | 77 | 已委托codec |
| `scalar.go` | 标量转换facade | 77 | 已委托scalar |
| `validation.go` | 请求验证器 | 70 | 已委托validation |

## 不做的事

1. 不否定动态生成，改的是生成边界
2. 不用 middleware 二次 decode 做 Argus 主路径
3. 不把 `DoubleArray` 继续暴露给新生成代码
4. 不为了“目录好看”继续搬文件，先把 generator/runtime/go-rpc-gateway 串起来
5. 不把 go-argus 具体类型写死进 `grpc-runtime`
