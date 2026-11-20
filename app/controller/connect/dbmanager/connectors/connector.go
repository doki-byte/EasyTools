package connectors

import (
	"EasyTools/app/controller/connect/dbmanager/app/model"
	"fmt"

	"github.com/jmoiron/sqlx"
)

const (
	Mysql        = "mysql"
	DorisLegacy  = "doris_legacy" // Apache Doris 等：旧版 FE 与 go-sql-driver v1.8+ 握手不兼容时使用
	Oracle       = "oracle"
	SQLite     = "sqlite"
	PostgreSQL = "postgresql"
	SQLServer  = "sqlserver"
	MariaDB    = "mariadb"
	DM         = "dm"       // 达梦数据库
	Kingbase   = "kingbase" // 人大金仓
	GBase      = "gbase"    // 南大通用
)

type Connector interface {
	Ping(config *model.DbConfig) error
	Connect(config *model.DbConfig) (*sqlx.DB, error)
	ShowDatabases(db *sqlx.DB) ([]string, error)
	Use(db *sqlx.DB, database string) error
	ShowTables(db *sqlx.DB, database string) (*[]Table, error)
	Column(db *sqlx.DB, params ...string) (*[]Column, error)
	Query(db *sqlx.DB, sqlStr string) (*Query, error)
	Count(db *sqlx.DB, sqlStr string) (int, error)
	Ddl(db *sqlx.DB, table string) (string, error)
	Execute(db *sqlx.DB, sqlStr string) (int, error)
}

type ConnectorFactory struct {
	mysqlConnector       MysqlConnector
	dorisLegacyConnector DorisLegacyConnector
	oracleConnector      OracleConnector
	sqLiteConnector     SQLiteConnector
	postgreSQLConnector PostgreSQLConnector
	sqlServerConnector  SQLServerConnector
	mariadbConnector    MariaDBConnector
	dmConnector         DMConnector
	kingbaseConnector   KingbaseConnector
	gbaseConnector      GBaseConnector
}

func (c ConnectorFactory) GetInstance(dbType string) (Connector, error) {
	var connector Connector
	switch dbType {
	case Mysql:
		connector = c.mysqlConnector
	case DorisLegacy:
		connector = c.dorisLegacyConnector
	case Oracle:
		connector = c.oracleConnector
	case SQLite:
		connector = c.sqLiteConnector
	case PostgreSQL:
		connector = c.postgreSQLConnector
	case SQLServer:
		connector = c.sqlServerConnector
	case MariaDB:
		connector = c.mariadbConnector
	case DM:
		connector = c.dmConnector
	case Kingbase:
		connector = c.kingbaseConnector
	case GBase:
		connector = c.gbaseConnector
	default:
		err := fmt.Errorf("database not support: %s", dbType)
		return nil, err
	}
	return connector, nil
}
