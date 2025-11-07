export namespace clause {
	
	export class Clause {
	    Name: string;
	    BeforeExpression: any;
	    AfterNameExpression: any;
	    AfterExpression: any;
	    Expression: any;
	
	    static createFrom(source: any = {}) {
	        return new Clause(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Name = source["Name"];
	        this.BeforeExpression = source["BeforeExpression"];
	        this.AfterNameExpression = source["AfterNameExpression"];
	        this.AfterExpression = source["AfterExpression"];
	        this.Expression = source["Expression"];
	    }
	}
	export class Expr {
	    SQL: string;
	    Vars: any[];
	    WithoutParentheses: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Expr(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.SQL = source["SQL"];
	        this.Vars = source["Vars"];
	        this.WithoutParentheses = source["WithoutParentheses"];
	    }
	}
	export class Where {
	    Exprs: any[];
	
	    static createFrom(source: any = {}) {
	        return new Where(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Exprs = source["Exprs"];
	    }
	}

}

export namespace config {
	
	export class Config {
	    Email: string;
	    FofaKey: string;
	    HunterKey: string;
	    QuakeKey: string;
	    Country: string;
	    Maxpage: string;
	    CoroutineCount: number;
	    LiveProxies: number;
	    AllProxies: number;
	    LiveProxyLists: string[];
	    Timeout: string;
	    SocksAddress: string;
	    FilePath: string;
	    Status: number;
	    Code: number;
	    Error: string;
	    GlobalProxy: string;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Email = source["Email"];
	        this.FofaKey = source["FofaKey"];
	        this.HunterKey = source["HunterKey"];
	        this.QuakeKey = source["QuakeKey"];
	        this.Country = source["Country"];
	        this.Maxpage = source["Maxpage"];
	        this.CoroutineCount = source["CoroutineCount"];
	        this.LiveProxies = source["LiveProxies"];
	        this.AllProxies = source["AllProxies"];
	        this.LiveProxyLists = source["LiveProxyLists"];
	        this.Timeout = source["Timeout"];
	        this.SocksAddress = source["SocksAddress"];
	        this.FilePath = source["FilePath"];
	        this.Status = source["Status"];
	        this.Code = source["Code"];
	        this.Error = source["Error"];
	        this.GlobalProxy = source["GlobalProxy"];
	    }
	}

}

export namespace controller {
	
	export class FileInfo {
	    name: string;
	    path: string;
	    isDir: boolean;
	    size: number;
	    // Go type: time
	    modified: any;
	    extension: string;
	
	    static createFrom(source: any = {}) {
	        return new FileInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.isDir = source["isDir"];
	        this.size = source["size"];
	        this.modified = this.convertValues(source["modified"], null);
	        this.extension = source["extension"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class JwtResult {
	    jwt_token: string;
	    header: Record<string, any>;
	    payload: Record<string, any>;
	    signature: string;
	    valid: boolean;
	    secret?: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new JwtResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.jwt_token = source["jwt_token"];
	        this.header = source["header"];
	        this.payload = source["payload"];
	        this.signature = source["signature"];
	        this.valid = source["valid"];
	        this.secret = source["secret"];
	        this.error = source["error"];
	    }
	}
	export class PasswordData {
	    id: number;
	    name: string;
	    method: string;
	    userId: string;
	    password: string;
	    level: string;
	
	    static createFrom(source: any = {}) {
	        return new PasswordData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.method = source["method"];
	        this.userId = source["userId"];
	        this.password = source["password"];
	        this.level = source["level"];
	    }
	}
	export class SiteItem {
	    id: number;
	    category: string;
	    title: string;
	    remark: string;
	    url: string;
	    icon?: string;
	
	    static createFrom(source: any = {}) {
	        return new SiteItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.category = source["category"];
	        this.title = source["title"];
	        this.remark = source["remark"];
	        this.url = source["url"];
	        this.icon = source["icon"];
	    }
	}
	export class SiteCategory {
	    title: string;
	    list: SiteItem[];
	
	    static createFrom(source: any = {}) {
	        return new SiteCategory(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.list = this.convertValues(source["list"], SiteItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class ToolsItem {
	    id: number;
	    category: string;
	    cmd: string;
	    param: string;
	    name: string;
	    path: string;
	    desc: string;
	    icon?: string;
	    terminal: number;
	
	    static createFrom(source: any = {}) {
	        return new ToolsItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.category = source["category"];
	        this.cmd = source["cmd"];
	        this.param = source["param"];
	        this.name = source["name"];
	        this.path = source["path"];
	        this.desc = source["desc"];
	        this.icon = source["icon"];
	        this.terminal = source["terminal"];
	    }
	}
	export class ToolsCategory {
	    title: string;
	    list: ToolsItem[];
	
	    static createFrom(source: any = {}) {
	        return new ToolsCategory(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.list = this.convertValues(source["list"], ToolsItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class UpdateUserItem {
	    username: string;
	    password: string;
	    OldPassword: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateUserItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.username = source["username"];
	        this.password = source["password"];
	        this.OldPassword = source["OldPassword"];
	    }
	}

}

export namespace define {
	
	export class Connection {
	    identity: string;
	    name: string;
	    addr: string;
	    port: string;
	    username: string;
	    password: string;
	    type: string;
	    ssh_addr: string;
	    ssh_port: string;
	    ssh_username: string;
	    ssh_password: string;
	
	    static createFrom(source: any = {}) {
	        return new Connection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.identity = source["identity"];
	        this.name = source["name"];
	        this.addr = source["addr"];
	        this.port = source["port"];
	        this.username = source["username"];
	        this.password = source["password"];
	        this.type = source["type"];
	        this.ssh_addr = source["ssh_addr"];
	        this.ssh_port = source["ssh_port"];
	        this.ssh_username = source["ssh_username"];
	        this.ssh_password = source["ssh_password"];
	    }
	}
	export class CreateKeyValueRequest {
	    conn_identity: string;
	    db: number;
	    key: string;
	    type: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateKeyValueRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.conn_identity = source["conn_identity"];
	        this.db = source["db"];
	        this.key = source["key"];
	        this.type = source["type"];
	    }
	}
	export class HashAddOrUpdateFieldRequest {
	    conn_identity: string;
	    db: number;
	    key: string;
	    field: string;
	    value: string;
	
	    static createFrom(source: any = {}) {
	        return new HashAddOrUpdateFieldRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.conn_identity = source["conn_identity"];
	        this.db = source["db"];
	        this.key = source["key"];
	        this.field = source["field"];
	        this.value = source["value"];
	    }
	}
	export class HashFieldDeleteRequest {
	    conn_identity: string;
	    db: number;
	    key: string;
	    field: string[];
	
	    static createFrom(source: any = {}) {
	        return new HashFieldDeleteRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.conn_identity = source["conn_identity"];
	        this.db = source["db"];
	        this.key = source["key"];
	        this.field = source["field"];
	    }
	}
	export class KeyListRequest {
	    conn_identity: string;
	    db: number;
	    keyword: string;
	
	    static createFrom(source: any = {}) {
	        return new KeyListRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.conn_identity = source["conn_identity"];
	        this.db = source["db"];
	        this.keyword = source["keyword"];
	    }
	}
	export class KeyValueRequest {
	    conn_identity: string;
	    db: number;
	    key: string;
	
	    static createFrom(source: any = {}) {
	        return new KeyValueRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.conn_identity = source["conn_identity"];
	        this.db = source["db"];
	        this.key = source["key"];
	    }
	}
	export class ListValueRequest {
	    conn_identity: string;
	    db: number;
	    key: string;
	    value: string;
	
	    static createFrom(source: any = {}) {
	        return new ListValueRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.conn_identity = source["conn_identity"];
	        this.db = source["db"];
	        this.key = source["key"];
	        this.value = source["value"];
	    }
	}
	export class SetValueRequest {
	    conn_identity: string;
	    db: number;
	    key: string;
	    value: string;
	
	    static createFrom(source: any = {}) {
	        return new SetValueRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.conn_identity = source["conn_identity"];
	        this.db = source["db"];
	        this.key = source["key"];
	        this.value = source["value"];
	    }
	}
	export class UpdateKeyValueRequest {
	    conn_identity: string;
	    db: number;
	    key: string;
	    ttl: number;
	    value: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateKeyValueRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.conn_identity = source["conn_identity"];
	        this.db = source["db"];
	        this.key = source["key"];
	        this.ttl = source["ttl"];
	        this.value = source["value"];
	    }
	}
	export class ZSetValueRequest {
	    conn_identity: string;
	    db: number;
	    key: string;
	    score: number;
	    member: any;
	
	    static createFrom(source: any = {}) {
	        return new ZSetValueRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.conn_identity = source["conn_identity"];
	        this.db = source["db"];
	        this.key = source["key"];
	        this.score = source["score"];
	        this.member = source["member"];
	    }
	}

}

export namespace gorm {
	
	export class join {
	    Name: string;
	    Conds: any[];
	    On?: clause.Where;
	    Selects: string[];
	    Omits: string[];
	    JoinType: string;
	
	    static createFrom(source: any = {}) {
	        return new join(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Name = source["Name"];
	        this.Conds = source["Conds"];
	        this.On = this.convertValues(source["On"], clause.Where);
	        this.Selects = source["Selects"];
	        this.Omits = source["Omits"];
	        this.JoinType = source["JoinType"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Statement {
	    SkipDefaultTransaction: boolean;
	    NamingStrategy: any;
	    FullSaveAssociations: boolean;
	    Logger: any;
	    DryRun: boolean;
	    PrepareStmt: boolean;
	    DisableAutomaticPing: boolean;
	    DisableForeignKeyConstraintWhenMigrating: boolean;
	    IgnoreRelationshipsWhenMigrating: boolean;
	    DisableNestedTransaction: boolean;
	    AllowGlobalUpdate: boolean;
	    QueryFields: boolean;
	    CreateBatchSize: number;
	    TranslateError: boolean;
	    PropagateUnscoped: boolean;
	    ClauseBuilders: Record<string, ClauseBuilder>;
	    ConnPool: any;
	    Dialector: any;
	    Plugins: Record<string, any>;
	    Error: any;
	    RowsAffected: number;
	    Statement?: Statement;
	    TableExpr?: clause.Expr;
	    Table: string;
	    Model: any;
	    Unscoped: boolean;
	    Dest: any;
	    // Go type: reflect
	    ReflectValue: any;
	    Clauses: Record<string, clause.Clause>;
	    BuildClauses: string[];
	    Distinct: boolean;
	    Selects: string[];
	    Omits: string[];
	    ColumnMapping: Record<string, string>;
	    Joins: join[];
	    Preloads: Record<string, Array<any>>;
	    // Go type: sync
	    Settings: any;
	    ConnPool: any;
	    Schema?: schema.Schema;
	    Context: any;
	    RaiseErrorOnNotFound: boolean;
	    SkipHooks: boolean;
	    // Go type: strings
	    SQL: any;
	    Vars: any[];
	    CurDestIndex: number;
	
	    static createFrom(source: any = {}) {
	        return new Statement(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.SkipDefaultTransaction = source["SkipDefaultTransaction"];
	        this.NamingStrategy = source["NamingStrategy"];
	        this.FullSaveAssociations = source["FullSaveAssociations"];
	        this.Logger = source["Logger"];
	        this.DryRun = source["DryRun"];
	        this.PrepareStmt = source["PrepareStmt"];
	        this.DisableAutomaticPing = source["DisableAutomaticPing"];
	        this.DisableForeignKeyConstraintWhenMigrating = source["DisableForeignKeyConstraintWhenMigrating"];
	        this.IgnoreRelationshipsWhenMigrating = source["IgnoreRelationshipsWhenMigrating"];
	        this.DisableNestedTransaction = source["DisableNestedTransaction"];
	        this.AllowGlobalUpdate = source["AllowGlobalUpdate"];
	        this.QueryFields = source["QueryFields"];
	        this.CreateBatchSize = source["CreateBatchSize"];
	        this.TranslateError = source["TranslateError"];
	        this.PropagateUnscoped = source["PropagateUnscoped"];
	        this.ClauseBuilders = source["ClauseBuilders"];
	        this.ConnPool = source["ConnPool"];
	        this.Dialector = source["Dialector"];
	        this.Plugins = source["Plugins"];
	        this.Error = source["Error"];
	        this.RowsAffected = source["RowsAffected"];
	        this.Statement = this.convertValues(source["Statement"], Statement);
	        this.TableExpr = this.convertValues(source["TableExpr"], clause.Expr);
	        this.Table = source["Table"];
	        this.Model = source["Model"];
	        this.Unscoped = source["Unscoped"];
	        this.Dest = source["Dest"];
	        this.ReflectValue = this.convertValues(source["ReflectValue"], null);
	        this.Clauses = this.convertValues(source["Clauses"], clause.Clause, true);
	        this.BuildClauses = source["BuildClauses"];
	        this.Distinct = source["Distinct"];
	        this.Selects = source["Selects"];
	        this.Omits = source["Omits"];
	        this.ColumnMapping = source["ColumnMapping"];
	        this.Joins = this.convertValues(source["Joins"], join);
	        this.Preloads = source["Preloads"];
	        this.Settings = this.convertValues(source["Settings"], null);
	        this.ConnPool = source["ConnPool"];
	        this.Schema = this.convertValues(source["Schema"], schema.Schema);
	        this.Context = source["Context"];
	        this.RaiseErrorOnNotFound = source["RaiseErrorOnNotFound"];
	        this.SkipHooks = source["SkipHooks"];
	        this.SQL = this.convertValues(source["SQL"], null);
	        this.Vars = source["Vars"];
	        this.CurDestIndex = source["CurDestIndex"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DB {
	    SkipDefaultTransaction: boolean;
	    NamingStrategy: any;
	    FullSaveAssociations: boolean;
	    Logger: any;
	    DryRun: boolean;
	    PrepareStmt: boolean;
	    DisableAutomaticPing: boolean;
	    DisableForeignKeyConstraintWhenMigrating: boolean;
	    IgnoreRelationshipsWhenMigrating: boolean;
	    DisableNestedTransaction: boolean;
	    AllowGlobalUpdate: boolean;
	    QueryFields: boolean;
	    CreateBatchSize: number;
	    TranslateError: boolean;
	    PropagateUnscoped: boolean;
	    ClauseBuilders: Record<string, ClauseBuilder>;
	    ConnPool: any;
	    Dialector: any;
	    Plugins: Record<string, any>;
	    Error: any;
	    RowsAffected: number;
	    Statement?: Statement;
	
	    static createFrom(source: any = {}) {
	        return new DB(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.SkipDefaultTransaction = source["SkipDefaultTransaction"];
	        this.NamingStrategy = source["NamingStrategy"];
	        this.FullSaveAssociations = source["FullSaveAssociations"];
	        this.Logger = source["Logger"];
	        this.DryRun = source["DryRun"];
	        this.PrepareStmt = source["PrepareStmt"];
	        this.DisableAutomaticPing = source["DisableAutomaticPing"];
	        this.DisableForeignKeyConstraintWhenMigrating = source["DisableForeignKeyConstraintWhenMigrating"];
	        this.IgnoreRelationshipsWhenMigrating = source["IgnoreRelationshipsWhenMigrating"];
	        this.DisableNestedTransaction = source["DisableNestedTransaction"];
	        this.AllowGlobalUpdate = source["AllowGlobalUpdate"];
	        this.QueryFields = source["QueryFields"];
	        this.CreateBatchSize = source["CreateBatchSize"];
	        this.TranslateError = source["TranslateError"];
	        this.PropagateUnscoped = source["PropagateUnscoped"];
	        this.ClauseBuilders = source["ClauseBuilders"];
	        this.ConnPool = source["ConnPool"];
	        this.Dialector = source["Dialector"];
	        this.Plugins = source["Plugins"];
	        this.Error = source["Error"];
	        this.RowsAffected = source["RowsAffected"];
	        this.Statement = this.convertValues(source["Statement"], Statement);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	

}

export namespace http {
	
	export class Client {
	    Transport: any;
	    Jar: any;
	    Timeout: number;
	
	    static createFrom(source: any = {}) {
	        return new Client(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Transport = source["Transport"];
	        this.Jar = source["Jar"];
	        this.Timeout = source["Timeout"];
	    }
	}

}

export namespace proxy {
	
	export class ProxyAuth {
	    username: string;
	    password: string;
	
	    static createFrom(source: any = {}) {
	        return new ProxyAuth(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.username = source["username"];
	        this.password = source["password"];
	    }
	}
	export class ProxyConfig {
	    type: string;
	    host: string;
	    port: string;
	    timeout: number;
	    auth?: ProxyAuth;
	
	    static createFrom(source: any = {}) {
	        return new ProxyConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.timeout = source["timeout"];
	        this.auth = this.convertValues(source["auth"], ProxyAuth);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GlobalProxyResponse {
	    config?: ProxyConfig;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new GlobalProxyResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.config = this.convertValues(source["config"], ProxyConfig);
	        this.enabled = source["enabled"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class ProxyStatus {
	    globalEnabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ProxyStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.globalEnabled = source["globalEnabled"];
	    }
	}
	export class Response {
	    Code: number;
	    Message: string;
	    Data: any;
	
	    static createFrom(source: any = {}) {
	        return new Response(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Code = source["Code"];
	        this.Message = source["Message"];
	        this.Data = source["Data"];
	    }
	}

}

export namespace reflect {
	
	export class StructField {
	    Name: string;
	    PkgPath: string;
	    Type: any;
	    Tag: string;
	    Offset: any;
	    Index: number[];
	    Anonymous: boolean;
	
	    static createFrom(source: any = {}) {
	        return new StructField(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Name = source["Name"];
	        this.PkgPath = source["PkgPath"];
	        this.Type = source["Type"];
	        this.Tag = source["Tag"];
	        this.Offset = source["Offset"];
	        this.Index = source["Index"];
	        this.Anonymous = source["Anonymous"];
	    }
	}

}

export namespace restmate {
	
	export class FormData {
	    id: string;
	    key: string;
	    value: string;
	    files: string[];
	    type: string;
	    active: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FormData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.key = source["key"];
	        this.value = source["value"];
	        this.files = source["files"];
	        this.type = source["type"];
	        this.active = source["active"];
	    }
	}
	export class Body {
	    bodyType: string;
	    bodyRaw: string;
	    formData: FormData[];
	
	    static createFrom(source: any = {}) {
	        return new Body(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bodyType = source["bodyType"];
	        this.bodyRaw = source["bodyRaw"];
	        this.formData = this.convertValues(source["formData"], FormData);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class JSResp {
	    success: boolean;
	    msg: string;
	    data?: any;
	
	    static createFrom(source: any = {}) {
	        return new JSResp(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.msg = source["msg"];
	        this.data = source["data"];
	    }
	}
	export class KeyValue {
	    id: string;
	    key: string;
	    value: string;
	    active: boolean;
	
	    static createFrom(source: any = {}) {
	        return new KeyValue(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.key = source["key"];
	        this.value = source["value"];
	        this.active = source["active"];
	    }
	}
	export class Request {
	    id: string;
	    name: string;
	    url: string;
	    method: string;
	    headers: KeyValue[];
	    params: KeyValue[];
	    body: Body;
	    coll_id: string;
	
	    static createFrom(source: any = {}) {
	        return new Request(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.url = source["url"];
	        this.method = source["method"];
	        this.headers = this.convertValues(source["headers"], KeyValue);
	        this.params = this.convertValues(source["params"], KeyValue);
	        this.body = this.convertValues(source["body"], Body);
	        this.coll_id = source["coll_id"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace schema {
	
	export class Reference {
	    PrimaryKey?: Field;
	    PrimaryValue: string;
	    ForeignKey?: Field;
	    OwnPrimaryKey: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Reference(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.PrimaryKey = this.convertValues(source["PrimaryKey"], Field);
	        this.PrimaryValue = source["PrimaryValue"];
	        this.ForeignKey = this.convertValues(source["ForeignKey"], Field);
	        this.OwnPrimaryKey = source["OwnPrimaryKey"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Polymorphic {
	    PolymorphicID?: Field;
	    PolymorphicType?: Field;
	    Value: string;
	
	    static createFrom(source: any = {}) {
	        return new Polymorphic(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.PolymorphicID = this.convertValues(source["PolymorphicID"], Field);
	        this.PolymorphicType = this.convertValues(source["PolymorphicType"], Field);
	        this.Value = source["Value"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Relationship {
	    Name: string;
	    Type: string;
	    Field?: Field;
	    Polymorphic?: Polymorphic;
	    References: Reference[];
	    Schema?: Schema;
	    FieldSchema?: Schema;
	    JoinTable?: Schema;
	
	    static createFrom(source: any = {}) {
	        return new Relationship(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Name = source["Name"];
	        this.Type = source["Type"];
	        this.Field = this.convertValues(source["Field"], Field);
	        this.Polymorphic = this.convertValues(source["Polymorphic"], Polymorphic);
	        this.References = this.convertValues(source["References"], Reference);
	        this.Schema = this.convertValues(source["Schema"], Schema);
	        this.FieldSchema = this.convertValues(source["FieldSchema"], Schema);
	        this.JoinTable = this.convertValues(source["JoinTable"], Schema);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Relationships {
	    HasOne: Relationship[];
	    BelongsTo: Relationship[];
	    HasMany: Relationship[];
	    Many2Many: Relationship[];
	    Relations: Record<string, Relationship>;
	    EmbeddedRelations: Record<string, Relationships>;
	
	    static createFrom(source: any = {}) {
	        return new Relationships(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.HasOne = this.convertValues(source["HasOne"], Relationship);
	        this.BelongsTo = this.convertValues(source["BelongsTo"], Relationship);
	        this.HasMany = this.convertValues(source["HasMany"], Relationship);
	        this.Many2Many = this.convertValues(source["Many2Many"], Relationship);
	        this.Relations = this.convertValues(source["Relations"], Relationship, true);
	        this.EmbeddedRelations = this.convertValues(source["EmbeddedRelations"], Relationships, true);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Schema {
	    Name: string;
	    ModelType: any;
	    Table: string;
	    PrioritizedPrimaryField?: Field;
	    DBNames: string[];
	    PrimaryFields: Field[];
	    PrimaryFieldDBNames: string[];
	    Fields: Field[];
	    FieldsByName: Record<string, Field>;
	    FieldsByBindName: Record<string, Field>;
	    FieldsByDBName: Record<string, Field>;
	    FieldsWithDefaultDBValue: Field[];
	    Relationships: Relationships;
	    CreateClauses: any[];
	    QueryClauses: any[];
	    UpdateClauses: any[];
	    DeleteClauses: any[];
	    BeforeCreate: boolean;
	    AfterCreate: boolean;
	    BeforeUpdate: boolean;
	    AfterUpdate: boolean;
	    BeforeDelete: boolean;
	    AfterDelete: boolean;
	    BeforeSave: boolean;
	    AfterSave: boolean;
	    AfterFind: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Schema(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Name = source["Name"];
	        this.ModelType = source["ModelType"];
	        this.Table = source["Table"];
	        this.PrioritizedPrimaryField = this.convertValues(source["PrioritizedPrimaryField"], Field);
	        this.DBNames = source["DBNames"];
	        this.PrimaryFields = this.convertValues(source["PrimaryFields"], Field);
	        this.PrimaryFieldDBNames = source["PrimaryFieldDBNames"];
	        this.Fields = this.convertValues(source["Fields"], Field);
	        this.FieldsByName = this.convertValues(source["FieldsByName"], Field, true);
	        this.FieldsByBindName = this.convertValues(source["FieldsByBindName"], Field, true);
	        this.FieldsByDBName = this.convertValues(source["FieldsByDBName"], Field, true);
	        this.FieldsWithDefaultDBValue = this.convertValues(source["FieldsWithDefaultDBValue"], Field);
	        this.Relationships = this.convertValues(source["Relationships"], Relationships);
	        this.CreateClauses = source["CreateClauses"];
	        this.QueryClauses = source["QueryClauses"];
	        this.UpdateClauses = source["UpdateClauses"];
	        this.DeleteClauses = source["DeleteClauses"];
	        this.BeforeCreate = source["BeforeCreate"];
	        this.AfterCreate = source["AfterCreate"];
	        this.BeforeUpdate = source["BeforeUpdate"];
	        this.AfterUpdate = source["AfterUpdate"];
	        this.BeforeDelete = source["BeforeDelete"];
	        this.AfterDelete = source["AfterDelete"];
	        this.BeforeSave = source["BeforeSave"];
	        this.AfterSave = source["AfterSave"];
	        this.AfterFind = source["AfterFind"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Field {
	    Name: string;
	    DBName: string;
	    BindNames: string[];
	    EmbeddedBindNames: string[];
	    DataType: string;
	    GORMDataType: string;
	    PrimaryKey: boolean;
	    AutoIncrement: boolean;
	    AutoIncrementIncrement: number;
	    Creatable: boolean;
	    Updatable: boolean;
	    Readable: boolean;
	    AutoCreateTime: number;
	    AutoUpdateTime: number;
	    HasDefaultValue: boolean;
	    DefaultValue: string;
	    DefaultValueInterface: any;
	    NotNull: boolean;
	    Unique: boolean;
	    Comment: string;
	    Size: number;
	    Precision: number;
	    Scale: number;
	    IgnoreMigration: boolean;
	    FieldType: any;
	    IndirectFieldType: any;
	    StructField: reflect.StructField;
	    Tag: string;
	    TagSettings: Record<string, string>;
	    Schema?: Schema;
	    EmbeddedSchema?: Schema;
	    OwnerSchema?: Schema;
	    Serializer: any;
	    NewValuePool: any;
	    UniqueIndex: string;
	
	    static createFrom(source: any = {}) {
	        return new Field(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Name = source["Name"];
	        this.DBName = source["DBName"];
	        this.BindNames = source["BindNames"];
	        this.EmbeddedBindNames = source["EmbeddedBindNames"];
	        this.DataType = source["DataType"];
	        this.GORMDataType = source["GORMDataType"];
	        this.PrimaryKey = source["PrimaryKey"];
	        this.AutoIncrement = source["AutoIncrement"];
	        this.AutoIncrementIncrement = source["AutoIncrementIncrement"];
	        this.Creatable = source["Creatable"];
	        this.Updatable = source["Updatable"];
	        this.Readable = source["Readable"];
	        this.AutoCreateTime = source["AutoCreateTime"];
	        this.AutoUpdateTime = source["AutoUpdateTime"];
	        this.HasDefaultValue = source["HasDefaultValue"];
	        this.DefaultValue = source["DefaultValue"];
	        this.DefaultValueInterface = source["DefaultValueInterface"];
	        this.NotNull = source["NotNull"];
	        this.Unique = source["Unique"];
	        this.Comment = source["Comment"];
	        this.Size = source["Size"];
	        this.Precision = source["Precision"];
	        this.Scale = source["Scale"];
	        this.IgnoreMigration = source["IgnoreMigration"];
	        this.FieldType = source["FieldType"];
	        this.IndirectFieldType = source["IndirectFieldType"];
	        this.StructField = this.convertValues(source["StructField"], reflect.StructField);
	        this.Tag = source["Tag"];
	        this.TagSettings = source["TagSettings"];
	        this.Schema = this.convertValues(source["Schema"], Schema);
	        this.EmbeddedSchema = this.convertValues(source["EmbeddedSchema"], Schema);
	        this.OwnerSchema = this.convertValues(source["OwnerSchema"], Schema);
	        this.Serializer = source["Serializer"];
	        this.NewValuePool = source["NewValuePool"];
	        this.UniqueIndex = source["UniqueIndex"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	

}

export namespace system {
	
	export class Base {
	    Ctx: any;
	
	    static createFrom(source: any = {}) {
	        return new Base(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Ctx = source["Ctx"];
	    }
	}
	export class  {
	    name: string;
	    browser_download_url: string;
	
	    static createFrom(source: any = {}) {
	        return new (source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.browser_download_url = source["browser_download_url"];
	    }
	}
	export class GitHubRelease {
	    tag_name: string;
	    html_url: string;
	    body: string;
	    assets: [];
	
	    static createFrom(source: any = {}) {
	        return new GitHubRelease(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tag_name = source["tag_name"];
	        this.html_url = source["html_url"];
	        this.body = source["body"];
	        this.assets = this.convertValues(source["assets"], );
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CheckResult {
	    hasUpdate: boolean;
	    currentVersion: string;
	    latestRelease?: GitHubRelease;
	
	    static createFrom(source: any = {}) {
	        return new CheckResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hasUpdate = source["hasUpdate"];
	        this.currentVersion = source["currentVersion"];
	        this.latestRelease = this.convertValues(source["latestRelease"], GitHubRelease);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DownloadResult {
	    error: boolean;
	    msg: string;
	
	    static createFrom(source: any = {}) {
	        return new DownloadResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.error = source["error"];
	        this.msg = source["msg"];
	    }
	}

}

export namespace unwxapp {
	
	export class VersionTaskStatus {
	    Number: string;
	    DecompileStatus: string;
	    MatchStatus: string;
	    Message: string;
	
	    static createFrom(source: any = {}) {
	        return new VersionTaskStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Number = source["Number"];
	        this.DecompileStatus = source["DecompileStatus"];
	        this.MatchStatus = source["MatchStatus"];
	        this.Message = source["Message"];
	    }
	}
	export class MiniAppInfo {
	    id: number;
	    app_id: string;
	    nickname: string;
	    username: string;
	    description: string;
	    avatar: string;
	    uses_count: string;
	    principal_name: string;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	
	    static createFrom(source: any = {}) {
	        return new MiniAppInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.app_id = source["app_id"];
	        this.nickname = source["nickname"];
	        this.username = source["username"];
	        this.description = source["description"];
	        this.avatar = source["avatar"];
	        this.uses_count = source["uses_count"];
	        this.principal_name = source["principal_name"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class InfoToFront {
	    AppID: string;
	    UpdateDate: string;
	    Info?: MiniAppInfo;
	    Versions: VersionTaskStatus[];
	
	    static createFrom(source: any = {}) {
	        return new InfoToFront(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.AppID = source["AppID"];
	        this.UpdateDate = source["UpdateDate"];
	        this.Info = this.convertValues(source["Info"], MiniAppInfo);
	        this.Versions = this.convertValues(source["Versions"], VersionTaskStatus);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	

}

