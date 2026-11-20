# mysqllegacydriver

内嵌 [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql) **v1.7.1** 源码（与仓库根 `go.mod` 中的 v1.8+ 官方模块并存）。

- 在 `init` 中以驱动名 **`mysql_doris_legacy`** 注册（见 `register.go`），**不**注册 `mysql`，避免与主程序使用的 `github.com/go-sql-driver/mysql` 冲突。
- 用途：部分 **Apache Doris** 旧版本无法正确处理 v1.8+ 握手里的默认连接属性，会导致 `invalid connection`；此类实例请在简练助手中选择数据库类型 **「Doris (旧版兼容)」**（`doris_legacy`）。

许可证：MPL-2.0，见同目录 `LICENSE`。
