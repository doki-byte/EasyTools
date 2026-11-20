// 与主工程 github.com/go-sql-driver/mysql v1.8+ 并存：此处内嵌 v1.7.1 源码（MPL-2.0，见 LICENSE），
// 以独立驱动名注册，避免与默认 "mysql" 驱动冲突；用于旧版 Apache Doris FE 9030 等不兼容连接属性的服务端。
package mysqllegacy

import "database/sql"

func init() {
	sql.Register("mysql_doris_legacy", &MySQLDriver{})
}
