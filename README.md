# 单词拼写挑战

一个可本地运行的 Go 1.22.12 单词拼写练习应用。项目内置简单和中等两组固定词库，支持中文或乱序字母提示，保存答案、用时、错词与得分，并提供历史统计和多人确认汇总。

## 环境

- Go 1.22.12 或兼容版本
- Node.js 20（仅用于前端构建）

## 运行

```bash
go run ./cmd/server
```

浏览器访问 `http://localhost:8080`。可通过 `PORT` 环境变量修改端口。

## 测试

普通全量测试：

```bash
CGO_ENABLED=0 go test -count=1 ./...
```

并发竞态验收：

```bash
CGO_ENABLED=1 go test -race -count=1 ./...
```

测试全部使用内存 fixture 和显式同步信号，不需要数据库或网络服务。

## 前端构建

```bash
cd web
npm install
npm run build
```

构建输出位于 `web/dist/`。Go 服务直接内嵌源前端资源，因此无需先构建前端即可运行。

## API

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| `GET` | `/api/challenges?difficulty=simple` | 获取固定挑战题目 |
| `POST` | `/api/answers` | 提交答案与用时 |
| `GET` | `/api/history` | 查看答题历史 |
| `GET` | `/api/wrong-words` | 查看错词回放 |
| `GET` | `/api/stats` | 查看汇总统计 |
| `POST` | `/api/reviews/{recordID}/confirmations` | 提交操作员确认 |
| `GET` | `/api/reviews/{recordID}` | 查看确认汇总 |
