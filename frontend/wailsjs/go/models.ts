export namespace assistive {
	
	export class GoogleQuery {
	    id: number;
	    category: string;
	    description: string;
	    command: string;
	
	    static createFrom(source: any = {}) {
	        return new GoogleQuery(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.category = source["category"];
	        this.description = source["description"];
	        this.command = source["command"];
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

}

export namespace browserScan {
	
	export class VuePrefixStat {
	    prefix: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new VuePrefixStat(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.prefix = source["prefix"];
	        this.count = source["count"];
	    }
	}
	export class VuePageAnalysis {
	    detectedBasePath: string;
	    commonPrefixes: VuePrefixStat[];
	
	    static createFrom(source: any = {}) {
	        return new VuePageAnalysis(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.detectedBasePath = source["detectedBasePath"];
	        this.commonPrefixes = this.convertValues(source["commonPrefixes"], VuePrefixStat);
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
	export class VueRouteItem {
	    name: string;
	    path: string;
	    meta: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new VueRouteItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.meta = source["meta"];
	    }
	}
	export class VueAnalysisResult {
	    vueDetected: boolean;
	    routerDetected: boolean;
	    vueVersion: string;
	    modifiedRoutes: VueRouteItem[];
	    allRoutes: VueRouteItem[];
	    routerBase: string;
	    pageAnalysis?: VuePageAnalysis;
	    currentPath: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new VueAnalysisResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.vueDetected = source["vueDetected"];
	        this.routerDetected = source["routerDetected"];
	        this.vueVersion = source["vueVersion"];
	        this.modifiedRoutes = this.convertValues(source["modifiedRoutes"], VueRouteItem);
	        this.allRoutes = this.convertValues(source["allRoutes"], VueRouteItem);
	        this.routerBase = source["routerBase"];
	        this.pageAnalysis = this.convertValues(source["pageAnalysis"], VuePageAnalysis);
	        this.currentPath = source["currentPath"];
	        this.error = source["error"];
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
	export class ExtractResults {
	    ips: string[];
	    ip_ports: string[];
	    domains: string[];
	    paths: string[];
	    incomplete_paths: string[];
	    urls: string[];
	    static_files: string[];
	    js_files: string[];
	    api_endpoints: string[];
	    jwt: string[];
	    secrets: string[];
	    mobiles: string[];
	    emails: string[];
	    sfz: string[];
	    algorithms: string[];
	    security_keys: string[];
	    passwords: string[];
	    usernames: string[];
	    aksk: string[];
	    app_ids: string[];
	    access_keys: string[];
	    api_keys: string[];
	    secret_keys: string[];
	    id_cards: string[];
	    vue_analysis?: VueAnalysisResult;
	
	    static createFrom(source: any = {}) {
	        return new ExtractResults(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ips = source["ips"];
	        this.ip_ports = source["ip_ports"];
	        this.domains = source["domains"];
	        this.paths = source["paths"];
	        this.incomplete_paths = source["incomplete_paths"];
	        this.urls = source["urls"];
	        this.static_files = source["static_files"];
	        this.js_files = source["js_files"];
	        this.api_endpoints = source["api_endpoints"];
	        this.jwt = source["jwt"];
	        this.secrets = source["secrets"];
	        this.mobiles = source["mobiles"];
	        this.emails = source["emails"];
	        this.sfz = source["sfz"];
	        this.algorithms = source["algorithms"];
	        this.security_keys = source["security_keys"];
	        this.passwords = source["passwords"];
	        this.usernames = source["usernames"];
	        this.aksk = source["aksk"];
	        this.app_ids = source["app_ids"];
	        this.access_keys = source["access_keys"];
	        this.api_keys = source["api_keys"];
	        this.secret_keys = source["secret_keys"];
	        this.id_cards = source["id_cards"];
	        this.vue_analysis = this.convertValues(source["vue_analysis"], VueAnalysisResult);
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
	export class ExtractedInfoItem {
	    rule: string;
	    value: string;
	
	    static createFrom(source: any = {}) {
	        return new ExtractedInfoItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rule = source["rule"];
	        this.value = source["value"];
	    }
	}
	export class LogEntry {
	    time: string;
	    level: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new LogEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.time = source["time"];
	        this.level = source["level"];
	        this.message = source["message"];
	    }
	}
	export class PacketLog {
	    url: string;
	    method: string;
	    status_code: number;
	    duration: string;
	    request_packet: string;
	    response_packet: string;
	    extracted_info: ExtractedInfoItem[];
	
	    static createFrom(source: any = {}) {
	        return new PacketLog(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.method = source["method"];
	        this.status_code = source["status_code"];
	        this.duration = source["duration"];
	        this.request_packet = source["request_packet"];
	        this.response_packet = source["response_packet"];
	        this.extracted_info = this.convertValues(source["extracted_info"], ExtractedInfoItem);
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
	export class ScanConfig {
	    wait_time: number;
	    extract_rules: string[];
	    api_routes: string[];
	    global_api_routes: string[];
	    allow_list: string[];
	    auto_test_api: boolean;
	    test_methods: string[];
	    worker_num: number;
	
	    static createFrom(source: any = {}) {
	        return new ScanConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.wait_time = source["wait_time"];
	        this.extract_rules = source["extract_rules"];
	        this.api_routes = source["api_routes"];
	        this.global_api_routes = source["global_api_routes"];
	        this.allow_list = source["allow_list"];
	        this.auto_test_api = source["auto_test_api"];
	        this.test_methods = source["test_methods"];
	        this.worker_num = source["worker_num"];
	    }
	}
	export class ScanTask {
	    id: string;
	    url: string;
	    status: string;
	    // Go type: time
	    start_time: any;
	    // Go type: time
	    end_time?: any;
	    results?: ExtractResults;
	    logs: LogEntry[];
	    config?: ScanConfig;
	    packet_logs: PacketLog[];
	
	    static createFrom(source: any = {}) {
	        return new ScanTask(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.url = source["url"];
	        this.status = source["status"];
	        this.start_time = this.convertValues(source["start_time"], null);
	        this.end_time = this.convertValues(source["end_time"], null);
	        this.results = this.convertValues(source["results"], ExtractResults);
	        this.logs = this.convertValues(source["logs"], LogEntry);
	        this.config = this.convertValues(source["config"], ScanConfig);
	        this.packet_logs = this.convertValues(source["packet_logs"], PacketLog);
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
	export class TaskBasicInfo {
	    id: string;
	    url: string;
	    status: string;
	    // Go type: time
	    start_time: any;
	    // Go type: time
	    end_time?: any;
	
	    static createFrom(source: any = {}) {
	        return new TaskBasicInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.url = source["url"];
	        this.status = source["status"];
	        this.start_time = this.convertValues(source["start_time"], null);
	        this.end_time = this.convertValues(source["end_time"], null);
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
	    Maxpage: string;
	    FofaSocks5Query: string;
	    FofaXuiQuery: string;
	    HunterSocks5Query: string;
	    HunterXuiQuery: string;
	    QuakeSocks5Query: string;
	    QuakeXuiQuery: string;
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
	    Xui: string;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Email = source["Email"];
	        this.FofaKey = source["FofaKey"];
	        this.HunterKey = source["HunterKey"];
	        this.QuakeKey = source["QuakeKey"];
	        this.Maxpage = source["Maxpage"];
	        this.FofaSocks5Query = source["FofaSocks5Query"];
	        this.FofaXuiQuery = source["FofaXuiQuery"];
	        this.HunterSocks5Query = source["HunterSocks5Query"];
	        this.HunterXuiQuery = source["HunterXuiQuery"];
	        this.QuakeSocks5Query = source["QuakeSocks5Query"];
	        this.QuakeXuiQuery = source["QuakeXuiQuery"];
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
	        this.Xui = source["Xui"];
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

}

export namespace crack {
	
	export class TaskProgress {
	    task_id: string;
	    total_targets: number;
	    current_target: number;
	    // Go type: time
	    start_time: any;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new TaskProgress(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.task_id = source["task_id"];
	        this.total_targets = source["total_targets"];
	        this.current_target = source["current_target"];
	        this.start_time = this.convertValues(source["start_time"], null);
	        this.status = source["status"];
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

export namespace define {
	
	export class Connection {
	    identity: string;
	    name: string;
	    addr: string;
	    port: string;
	    username: string;
	    password: string;
	    type: string;
	    db_type: string;
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
	        this.db_type = source["db_type"];
	        this.ssh_addr = source["ssh_addr"];
	        this.ssh_port = source["ssh_port"];
	        this.ssh_username = source["ssh_username"];
	        this.ssh_password = source["ssh_password"];
	    }
	}
	export class CreateKeyValueRequest {
	    conn_identity: string;
	    db: any;
	    key: string;
	    type: string;
	    value: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateKeyValueRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.conn_identity = source["conn_identity"];
	        this.db = source["db"];
	        this.key = source["key"];
	        this.type = source["type"];
	        this.value = source["value"];
	    }
	}
	export class HashAddOrUpdateFieldRequest {
	    conn_identity: string;
	    db: any;
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
	    db: any;
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
	    db: any;
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
	    db: any;
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
	    db: any;
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
	export class MongoDocumentDeleteRequest {
	    conn_identity: string;
	    db: any;
	    key: string;
	    doc_id: string;
	
	    static createFrom(source: any = {}) {
	        return new MongoDocumentDeleteRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.conn_identity = source["conn_identity"];
	        this.db = source["db"];
	        this.key = source["key"];
	        this.doc_id = source["doc_id"];
	    }
	}
	export class SetValueRequest {
	    conn_identity: string;
	    db: any;
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
	    db: any;
	    key: string;
	    ttl: number;
	    value: string;
	    old_value: string;
	
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
	        this.old_value = source["old_value"];
	    }
	}
	export class ZSetValueRequest {
	    conn_identity: string;
	    db: any;
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

export namespace domainCollection {
	
	export class CollectionConfig {
	    dict_file: string;
	    dict_content: string[];
	    ai_provider: string;
	    ai_api_key: string;
	    ai_api_base: string;
	    ai_config_id: string;
	    sample_domains: string[];
	    generate_count: number;
	    enable_space_mapping: boolean;
	    space_mapping_platforms: string[];
	    threads: number;
	    timeout: number;
	    verify_dns: boolean;
	    verify_http: boolean;
	    dns_server: string;
	    wildcard: boolean;
	    ports: string;
	    filter_ip: string;
	    fingerprint: boolean;
	    poc_detection: boolean;
	    api_detection: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CollectionConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dict_file = source["dict_file"];
	        this.dict_content = source["dict_content"];
	        this.ai_provider = source["ai_provider"];
	        this.ai_api_key = source["ai_api_key"];
	        this.ai_api_base = source["ai_api_base"];
	        this.ai_config_id = source["ai_config_id"];
	        this.sample_domains = source["sample_domains"];
	        this.generate_count = source["generate_count"];
	        this.enable_space_mapping = source["enable_space_mapping"];
	        this.space_mapping_platforms = source["space_mapping_platforms"];
	        this.threads = source["threads"];
	        this.timeout = source["timeout"];
	        this.verify_dns = source["verify_dns"];
	        this.verify_http = source["verify_http"];
	        this.dns_server = source["dns_server"];
	        this.wildcard = source["wildcard"];
	        this.ports = source["ports"];
	        this.filter_ip = source["filter_ip"];
	        this.fingerprint = source["fingerprint"];
	        this.poc_detection = source["poc_detection"];
	        this.api_detection = source["api_detection"];
	    }
	}
	export class DomainInfo {
	    domain: string;
	    ips: string[];
	    status: string;
	    http_status: number;
	    http_title: string;
	    found_time: string;
	    source: string;
	    platform: string;
	
	    static createFrom(source: any = {}) {
	        return new DomainInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.domain = source["domain"];
	        this.ips = source["ips"];
	        this.status = source["status"];
	        this.http_status = source["http_status"];
	        this.http_title = source["http_title"];
	        this.found_time = source["found_time"];
	        this.source = source["source"];
	        this.platform = source["platform"];
	    }
	}
	export class CollectionResults {
	    total_generated: number;
	    valid_domains: DomainInfo[];
	    invalid_domains: string[];
	
	    static createFrom(source: any = {}) {
	        return new CollectionResults(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total_generated = source["total_generated"];
	        this.valid_domains = this.convertValues(source["valid_domains"], DomainInfo);
	        this.invalid_domains = source["invalid_domains"];
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
	export class LogEntry {
	    time: string;
	    level: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new LogEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.time = source["time"];
	        this.level = source["level"];
	        this.message = source["message"];
	    }
	}
	export class CollectionTask {
	    id: string;
	    name: string;
	    root_domain: string;
	    mode: string;
	    status: string;
	    // Go type: time
	    start_time: any;
	    // Go type: time
	    end_time?: any;
	    config?: CollectionConfig;
	    results?: CollectionResults;
	    logs: LogEntry[];
	
	    static createFrom(source: any = {}) {
	        return new CollectionTask(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.root_domain = source["root_domain"];
	        this.mode = source["mode"];
	        this.status = source["status"];
	        this.start_time = this.convertValues(source["start_time"], null);
	        this.end_time = this.convertValues(source["end_time"], null);
	        this.config = this.convertValues(source["config"], CollectionConfig);
	        this.results = this.convertValues(source["results"], CollectionResults);
	        this.logs = this.convertValues(source["logs"], LogEntry);
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

export namespace druid {
	
	export class BasicInfo {
	    javaClassPath: string;
	    javaVMName: string;
	    javaVersion: string;
	
	    static createFrom(source: any = {}) {
	        return new BasicInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.javaClassPath = source["javaClassPath"];
	        this.javaVMName = source["javaVMName"];
	        this.javaVersion = source["javaVersion"];
	    }
	}
	export class JDBCInfo {
	    userName: string;
	    jdbcUrl: string;
	    javaClassPath: string;
	    javaVMName: string;
	    javaVersion: string;
	
	    static createFrom(source: any = {}) {
	        return new JDBCInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.userName = source["userName"];
	        this.jdbcUrl = source["jdbcUrl"];
	        this.javaClassPath = source["javaClassPath"];
	        this.javaVMName = source["javaVMName"];
	        this.javaVersion = source["javaVersion"];
	    }
	}
	export class DruidResult {
	    success: boolean;
	    message: string;
	    sessions: string[];
	    sqls: string[];
	    uris: string[];
	    jdbcInfo: JDBCInfo;
	    basicInfo: BasicInfo;
	
	    static createFrom(source: any = {}) {
	        return new DruidResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.message = source["message"];
	        this.sessions = source["sessions"];
	        this.sqls = source["sqls"];
	        this.uris = source["uris"];
	        this.jdbcInfo = this.convertValues(source["jdbcInfo"], JDBCInfo);
	        this.basicInfo = this.convertValues(source["basicInfo"], BasicInfo);
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

export namespace fileDumper {
	
	export class DSStoreBatchScanRequest {
	    urls: string[];
	    outputDir: string;
	    jobs: number;
	    timeout: number;
	    retry: number;
	
	    static createFrom(source: any = {}) {
	        return new DSStoreBatchScanRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.urls = source["urls"];
	        this.outputDir = source["outputDir"];
	        this.jobs = source["jobs"];
	        this.timeout = source["timeout"];
	        this.retry = source["retry"];
	    }
	}
	export class DSStoreScanResult {
	    url: string;
	    success: boolean;
	    message: string;
	    outputDir?: string;
	    timestamp?: string;
	
	    static createFrom(source: any = {}) {
	        return new DSStoreScanResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.success = source["success"];
	        this.message = source["message"];
	        this.outputDir = source["outputDir"];
	        this.timestamp = source["timestamp"];
	    }
	}
	export class GitBatchScanRequest {
	    urls: string[];
	    outputDir: string;
	    jobs: number;
	    timeout: number;
	    retry: number;
	    cancelToken?: string;
	
	    static createFrom(source: any = {}) {
	        return new GitBatchScanRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.urls = source["urls"];
	        this.outputDir = source["outputDir"];
	        this.jobs = source["jobs"];
	        this.timeout = source["timeout"];
	        this.retry = source["retry"];
	        this.cancelToken = source["cancelToken"];
	    }
	}
	export class GitScanResult {
	    url: string;
	    success: boolean;
	    message: string;
	    outputDir?: string;
	    timestamp?: string;
	
	    static createFrom(source: any = {}) {
	        return new GitScanResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.success = source["success"];
	        this.message = source["message"];
	        this.outputDir = source["outputDir"];
	        this.timestamp = source["timestamp"];
	    }
	}
	export class HgBatchScanRequest {
	    urls: string[];
	    outputDir: string;
	    jobs: number;
	    timeout: number;
	    retry: number;
	    taskId?: string;
	
	    static createFrom(source: any = {}) {
	        return new HgBatchScanRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.urls = source["urls"];
	        this.outputDir = source["outputDir"];
	        this.jobs = source["jobs"];
	        this.timeout = source["timeout"];
	        this.retry = source["retry"];
	        this.taskId = source["taskId"];
	    }
	}
	export class HgRetryScanRequest {
	    url: string;
	    outputDir: string;
	    jobs: number;
	    timeout: number;
	    retry: number;
	    taskId?: string;
	
	    static createFrom(source: any = {}) {
	        return new HgRetryScanRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.outputDir = source["outputDir"];
	        this.jobs = source["jobs"];
	        this.timeout = source["timeout"];
	        this.retry = source["retry"];
	        this.taskId = source["taskId"];
	    }
	}
	export class HgScanResult {
	    url: string;
	    success: boolean;
	    message: string;
	    outputDir?: string;
	    timestamp?: string;
	
	    static createFrom(source: any = {}) {
	        return new HgScanResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.success = source["success"];
	        this.message = source["message"];
	        this.outputDir = source["outputDir"];
	        this.timestamp = source["timestamp"];
	    }
	}
	export class IndexBatchScanRequest {
	    urls: string[];
	    outputDir: string;
	    jobs: number;
	    timeout: number;
	    retry: number;
	    cancelToken?: string;
	    headers?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new IndexBatchScanRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.urls = source["urls"];
	        this.outputDir = source["outputDir"];
	        this.jobs = source["jobs"];
	        this.timeout = source["timeout"];
	        this.retry = source["retry"];
	        this.cancelToken = source["cancelToken"];
	        this.headers = source["headers"];
	    }
	}
	export class IndexScanResult {
	    url: string;
	    success: boolean;
	    message: string;
	    outputDir?: string;
	    timestamp?: string;
	
	    static createFrom(source: any = {}) {
	        return new IndexScanResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.success = source["success"];
	        this.message = source["message"];
	        this.outputDir = source["outputDir"];
	        this.timestamp = source["timestamp"];
	    }
	}
	export class RetryDSStoreScanRequest {
	    url: string;
	    outputDir: string;
	    jobs: number;
	    timeout: number;
	    retry: number;
	
	    static createFrom(source: any = {}) {
	        return new RetryDSStoreScanRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.outputDir = source["outputDir"];
	        this.jobs = source["jobs"];
	        this.timeout = source["timeout"];
	        this.retry = source["retry"];
	    }
	}
	export class RetryGitScanRequest {
	    url: string;
	    outputDir: string;
	    jobs: number;
	    timeout: number;
	    retry: number;
	
	    static createFrom(source: any = {}) {
	        return new RetryGitScanRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.outputDir = source["outputDir"];
	        this.jobs = source["jobs"];
	        this.timeout = source["timeout"];
	        this.retry = source["retry"];
	    }
	}
	export class RetryIndexScanRequest {
	    url: string;
	    outputDir: string;
	    jobs: number;
	    timeout: number;
	    retry: number;
	    headers?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new RetryIndexScanRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.outputDir = source["outputDir"];
	        this.jobs = source["jobs"];
	        this.timeout = source["timeout"];
	        this.retry = source["retry"];
	        this.headers = source["headers"];
	    }
	}
	export class RetrySvnScanRequest {
	    url: string;
	    outputDir: string;
	    jobs: number;
	    timeout: number;
	    retry: number;
	
	    static createFrom(source: any = {}) {
	        return new RetrySvnScanRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.outputDir = source["outputDir"];
	        this.jobs = source["jobs"];
	        this.timeout = source["timeout"];
	        this.retry = source["retry"];
	    }
	}
	export class SvnBatchScanRequest {
	    urls: string[];
	    outputDir: string;
	    jobs: number;
	    timeout: number;
	    retry: number;
	
	    static createFrom(source: any = {}) {
	        return new SvnBatchScanRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.urls = source["urls"];
	        this.outputDir = source["outputDir"];
	        this.jobs = source["jobs"];
	        this.timeout = source["timeout"];
	        this.retry = source["retry"];
	    }
	}
	export class SvnScanResult {
	    url: string;
	    success: boolean;
	    message: string;
	    outputDir?: string;
	    timestamp?: string;
	
	    static createFrom(source: any = {}) {
	        return new SvnScanResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.success = source["success"];
	        this.message = source["message"];
	        this.outputDir = source["outputDir"];
	        this.timestamp = source["timestamp"];
	    }
	}

}

export namespace finger {
	
	export class fingerRule {
	    cms: string;
	    method: string;
	    location: string;
	    keyword: string[];
	
	    static createFrom(source: any = {}) {
	        return new fingerRule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cms = source["cms"];
	        this.method = source["method"];
	        this.location = source["location"];
	        this.keyword = source["keyword"];
	    }
	}

}

export namespace gin {
	
	export class Error {
	    Err: any;
	    Type: number;
	    Meta: any;
	
	    static createFrom(source: any = {}) {
	        return new Error(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Err = source["Err"];
	        this.Type = source["Type"];
	        this.Meta = source["Meta"];
	    }
	}
	export class Param {
	    Key: string;
	    Value: string;
	
	    static createFrom(source: any = {}) {
	        return new Param(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Key = source["Key"];
	        this.Value = source["Value"];
	    }
	}
	export class Context {
	    Request?: http.Request;
	    Writer: any;
	    Params: Param[];
	    Keys: Record<string, any>;
	    Errors: Error[];
	    Accepted: string[];
	
	    static createFrom(source: any = {}) {
	        return new Context(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Request = this.convertValues(source["Request"], http.Request);
	        this.Writer = source["Writer"];
	        this.Params = this.convertValues(source["Params"], Param);
	        this.Keys = source["Keys"];
	        this.Errors = this.convertValues(source["Errors"], Error);
	        this.Accepted = source["Accepted"];
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
	export class Response {
	    Status: string;
	    StatusCode: number;
	    Proto: string;
	    ProtoMajor: number;
	    ProtoMinor: number;
	    Header: Record<string, Array<string>>;
	    Body: any;
	    ContentLength: number;
	    TransferEncoding: string[];
	    Close: boolean;
	    Uncompressed: boolean;
	    Trailer: Record<string, Array<string>>;
	    Request?: Request;
	    TLS?: tls.ConnectionState;
	
	    static createFrom(source: any = {}) {
	        return new Response(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Status = source["Status"];
	        this.StatusCode = source["StatusCode"];
	        this.Proto = source["Proto"];
	        this.ProtoMajor = source["ProtoMajor"];
	        this.ProtoMinor = source["ProtoMinor"];
	        this.Header = source["Header"];
	        this.Body = source["Body"];
	        this.ContentLength = source["ContentLength"];
	        this.TransferEncoding = source["TransferEncoding"];
	        this.Close = source["Close"];
	        this.Uncompressed = source["Uncompressed"];
	        this.Trailer = source["Trailer"];
	        this.Request = this.convertValues(source["Request"], Request);
	        this.TLS = this.convertValues(source["TLS"], tls.ConnectionState);
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
	export class Request {
	    Method: string;
	    URL?: url.URL;
	    Proto: string;
	    ProtoMajor: number;
	    ProtoMinor: number;
	    Header: Record<string, Array<string>>;
	    Body: any;
	    ContentLength: number;
	    TransferEncoding: string[];
	    Close: boolean;
	    Host: string;
	    Form: Record<string, Array<string>>;
	    PostForm: Record<string, Array<string>>;
	    MultipartForm?: multipart.Form;
	    Trailer: Record<string, Array<string>>;
	    RemoteAddr: string;
	    RequestURI: string;
	    TLS?: tls.ConnectionState;
	    Response?: Response;
	    Pattern: string;
	
	    static createFrom(source: any = {}) {
	        return new Request(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Method = source["Method"];
	        this.URL = this.convertValues(source["URL"], url.URL);
	        this.Proto = source["Proto"];
	        this.ProtoMajor = source["ProtoMajor"];
	        this.ProtoMinor = source["ProtoMinor"];
	        this.Header = source["Header"];
	        this.Body = source["Body"];
	        this.ContentLength = source["ContentLength"];
	        this.TransferEncoding = source["TransferEncoding"];
	        this.Close = source["Close"];
	        this.Host = source["Host"];
	        this.Form = source["Form"];
	        this.PostForm = source["PostForm"];
	        this.MultipartForm = this.convertValues(source["MultipartForm"], multipart.Form);
	        this.Trailer = source["Trailer"];
	        this.RemoteAddr = source["RemoteAddr"];
	        this.RequestURI = source["RequestURI"];
	        this.TLS = this.convertValues(source["TLS"], tls.ConnectionState);
	        this.Response = this.convertValues(source["Response"], Response);
	        this.Pattern = source["Pattern"];
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

export namespace infoDeal {
	
	export class AuthorizedIPQueryRequest {
	    ip: string;
	    useAuthorized: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AuthorizedIPQueryRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ip = source["ip"];
	        this.useAuthorized = source["useAuthorized"];
	    }
	}
	export class BatchQueryConfig {
	    concurrency: number;
	    timeout: number;
	
	    static createFrom(source: any = {}) {
	        return new BatchQueryConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.concurrency = source["concurrency"];
	        this.timeout = source["timeout"];
	    }
	}
	export class IPResult {
	    original: string;
	    ip: string;
	    location: string;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new IPResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.original = source["original"];
	        this.ip = source["ip"];
	        this.location = source["location"];
	        this.status = source["status"];
	    }
	}
	export class IPExportRequest {
	    results: IPResult[];
	    fileName?: string;
	    timestamp?: string;
	
	    static createFrom(source: any = {}) {
	        return new IPExportRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.results = this.convertValues(source["results"], IPResult);
	        this.fileName = source["fileName"];
	        this.timestamp = source["timestamp"];
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
	export class IPQueryRequest {
	    targets: string[];
	
	    static createFrom(source: any = {}) {
	        return new IPQueryRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.targets = source["targets"];
	    }
	}
	export class IPQueryResponse {
	    success: boolean;
	    original?: string;
	    ip?: string;
	    location?: string;
	    detail?: string;
	    lat?: number;
	    lng?: number;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new IPQueryResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.original = source["original"];
	        this.ip = source["ip"];
	        this.location = source["location"];
	        this.detail = source["detail"];
	        this.lat = source["lat"];
	        this.lng = source["lng"];
	        this.error = source["error"];
	    }
	}

}

export namespace jwt {
	
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
	export class VulnResponse {
	    jwt: string;
	    header: Record<string, any>;
	    payload: Record<string, any>;
	    signature: string;
	    description: string;
	    test_vector: string;
	    mitigation: string;
	    severity: string;
	    error?: string;
	    data_source?: string;
	
	    static createFrom(source: any = {}) {
	        return new VulnResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.jwt = source["jwt"];
	        this.header = source["header"];
	        this.payload = source["payload"];
	        this.signature = source["signature"];
	        this.description = source["description"];
	        this.test_vector = source["test_vector"];
	        this.mitigation = source["mitigation"];
	        this.severity = source["severity"];
	        this.error = source["error"];
	        this.data_source = source["data_source"];
	    }
	}

}

export namespace model {
	
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
	export class ProtocolConnection {
	    id: number;
	    protocol: string;
	    host: string;
	    port: string;
	    username: string;
	    password: string;
	    status: string;
	    message: string;
	    credential: string;
	    vuln_url: string;
	    description: string;
	    command_result: string;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	
	    static createFrom(source: any = {}) {
	        return new ProtocolConnection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.protocol = source["protocol"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.username = source["username"];
	        this.password = source["password"];
	        this.status = source["status"];
	        this.message = source["message"];
	        this.credential = source["credential"];
	        this.vuln_url = source["vuln_url"];
	        this.description = source["description"];
	        this.command_result = source["command_result"];
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
	export class VulnerabilityInfo {
	    id: number;
	    task_id: string;
	    template_id: string;
	    vuln_name: string;
	    protocol: string;
	    severity: string;
	    vuln_url: string;
	    extract: string;
	    request: string;
	    response: string;
	    description: string;
	    reference: string;
	    response_time: string;
	
	    static createFrom(source: any = {}) {
	        return new VulnerabilityInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.task_id = source["task_id"];
	        this.template_id = source["template_id"];
	        this.vuln_name = source["vuln_name"];
	        this.protocol = source["protocol"];
	        this.severity = source["severity"];
	        this.vuln_url = source["vuln_url"];
	        this.extract = source["extract"];
	        this.request = source["request"];
	        this.response = source["response"];
	        this.description = source["description"];
	        this.reference = source["reference"];
	        this.response_time = source["response_time"];
	    }
	}

}

export namespace multipart {
	
	export class FileHeader {
	    Filename: string;
	    Header: Record<string, Array<string>>;
	    Size: number;
	
	    static createFrom(source: any = {}) {
	        return new FileHeader(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Filename = source["Filename"];
	        this.Header = source["Header"];
	        this.Size = source["Size"];
	    }
	}
	export class Form {
	    Value: Record<string, Array<string>>;
	    File: Record<string, Array<FileHeader>>;
	
	    static createFrom(source: any = {}) {
	        return new Form(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Value = source["Value"];
	        this.File = this.convertValues(source["File"], Array<FileHeader>, true);
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

export namespace net {
	
	export class IPNet {
	    IP: number[];
	    Mask: number[];
	
	    static createFrom(source: any = {}) {
	        return new IPNet(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.IP = source["IP"];
	        this.Mask = source["Mask"];
	    }
	}

}

export namespace pcap {
	
	export class AnalysisConfig {
	    filePath: string;
	    flagRegex: string;
	    sqlRegex: string;
	    credentialRegex: string;
	    keyRegex: string;
	    extractFiles: boolean;
	    detectFlags: boolean;
	    detectSQL: boolean;
	    detectCredentials: boolean;
	    detectKeys: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AnalysisConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filePath = source["filePath"];
	        this.flagRegex = source["flagRegex"];
	        this.sqlRegex = source["sqlRegex"];
	        this.credentialRegex = source["credentialRegex"];
	        this.keyRegex = source["keyRegex"];
	        this.extractFiles = source["extractFiles"];
	        this.detectFlags = source["detectFlags"];
	        this.detectSQL = source["detectSQL"];
	        this.detectCredentials = source["detectCredentials"];
	        this.detectKeys = source["detectKeys"];
	    }
	}
	export class AnalysisStatus {
	    isProcessing: boolean;
	    currentTask: string;
	    packets: number;
	    matches: number;
	    extractedFiles: number;
	    progress: string;
	
	    static createFrom(source: any = {}) {
	        return new AnalysisStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isProcessing = source["isProcessing"];
	        this.currentTask = source["currentTask"];
	        this.packets = source["packets"];
	        this.matches = source["matches"];
	        this.extractedFiles = source["extractedFiles"];
	        this.progress = source["progress"];
	    }
	}

}

export namespace pkix {
	
	export class AttributeTypeAndValue {
	    Type: number[];
	    Value: any;
	
	    static createFrom(source: any = {}) {
	        return new AttributeTypeAndValue(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Type = source["Type"];
	        this.Value = source["Value"];
	    }
	}
	export class Extension {
	    Id: number[];
	    Critical: boolean;
	    Value: number[];
	
	    static createFrom(source: any = {}) {
	        return new Extension(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Id = source["Id"];
	        this.Critical = source["Critical"];
	        this.Value = source["Value"];
	    }
	}
	export class Name {
	    Country: string[];
	    Organization: string[];
	    OrganizationalUnit: string[];
	    Locality: string[];
	    Province: string[];
	    StreetAddress: string[];
	    PostalCode: string[];
	    SerialNumber: string;
	    CommonName: string;
	    Names: AttributeTypeAndValue[];
	    ExtraNames: AttributeTypeAndValue[];
	
	    static createFrom(source: any = {}) {
	        return new Name(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Country = source["Country"];
	        this.Organization = source["Organization"];
	        this.OrganizationalUnit = source["OrganizationalUnit"];
	        this.Locality = source["Locality"];
	        this.Province = source["Province"];
	        this.StreetAddress = source["StreetAddress"];
	        this.PostalCode = source["PostalCode"];
	        this.SerialNumber = source["SerialNumber"];
	        this.CommonName = source["CommonName"];
	        this.Names = this.convertValues(source["Names"], AttributeTypeAndValue);
	        this.ExtraNames = this.convertValues(source["ExtraNames"], AttributeTypeAndValue);
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
	    systemEnabled: boolean;
	    enabledModules: number;
	    disabledModules: number;
	    totalModules: number;
	
	    static createFrom(source: any = {}) {
	        return new ProxyStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.globalEnabled = source["globalEnabled"];
	        this.systemEnabled = source["systemEnabled"];
	        this.enabledModules = source["enabledModules"];
	        this.disabledModules = source["disabledModules"];
	        this.totalModules = source["totalModules"];
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
	export class SystemProxyConfig {
	    enabled: boolean;
	    host: string;
	    port: string;
	    type: string;
	    override?: string;
	
	    static createFrom(source: any = {}) {
	        return new SystemProxyConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.type = source["type"];
	        this.override = source["override"];
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
	export class UploadBypassRequest {
	    requestPacket: string;
	    language: string;
	    forceProtocol: string;
	    proxyHost: string;
	    proxyPort: string;
	    payloads: string[];
	    concurrency: number;
	    retryCount: number;
	
	    static createFrom(source: any = {}) {
	        return new UploadBypassRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.requestPacket = source["requestPacket"];
	        this.language = source["language"];
	        this.forceProtocol = source["forceProtocol"];
	        this.proxyHost = source["proxyHost"];
	        this.proxyPort = source["proxyPort"];
	        this.payloads = source["payloads"];
	        this.concurrency = source["concurrency"];
	        this.retryCount = source["retryCount"];
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
	
	export class PersonItem {
	    name: string;
	    src: string;
	    url?: string;
	
	    static createFrom(source: any = {}) {
	        return new PersonItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.src = source["src"];
	        this.url = source["url"];
	    }
	}
	export class UpdateItem {
	    date: string;
	    content: string;
	    isNew: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UpdateItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.content = source["content"];
	        this.isNew = source["isNew"];
	    }
	}
	export class AboutInfo {
	    updates: UpdateItem[];
	    specialThanks: PersonItem[];
	    friendlySponsorship: PersonItem[];
	    latestVersion?: string;
	
	    static createFrom(source: any = {}) {
	        return new AboutInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.updates = this.convertValues(source["updates"], UpdateItem);
	        this.specialThanks = this.convertValues(source["specialThanks"], PersonItem);
	        this.friendlySponsorship = this.convertValues(source["friendlySponsorship"], PersonItem);
	        this.latestVersion = source["latestVersion"];
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
	export class FileItem {
	    name: string;
	    path: string;
	    isDir: boolean;
	    size: number;
	    modified: string;
	
	    static createFrom(source: any = {}) {
	        return new FileItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.isDir = source["isDir"];
	        this.size = source["size"];
	        this.modified = source["modified"];
	    }
	}
	
	
	export class PortConfig {
	    mainPort: number;
	    sshPort: number;
	    ftpPort: number;
	    fuzzPort: number;
	    dbManagerPort: number;
	
	    static createFrom(source: any = {}) {
	        return new PortConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mainPort = source["mainPort"];
	        this.sshPort = source["sshPort"];
	        this.ftpPort = source["ftpPort"];
	        this.fuzzPort = source["fuzzPort"];
	        this.dbManagerPort = source["dbManagerPort"];
	    }
	}
	export class SoftwareItem {
	    id: string;
	    name: string;
	    description: string;
	    latestVersion: string;
	    toolType: string;
	    toolLanguage: string;
	    lastUpdated: string;
	    author: string;
	    homepage: string;
	    isNew: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SoftwareItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.latestVersion = source["latestVersion"];
	        this.toolType = source["toolType"];
	        this.toolLanguage = source["toolLanguage"];
	        this.lastUpdated = source["lastUpdated"];
	        this.author = source["author"];
	        this.homepage = source["homepage"];
	        this.isNew = source["isNew"];
	    }
	}
	export class SoftwareCatalog {
	    catalogVersion: string;
	    updatedAt: string;
	    source?: string;
	    items: SoftwareItem[];
	
	    static createFrom(source: any = {}) {
	        return new SoftwareCatalog(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.catalogVersion = source["catalogVersion"];
	        this.updatedAt = source["updatedAt"];
	        this.source = source["source"];
	        this.items = this.convertValues(source["items"], SoftwareItem);
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
	
	export class SoftwareSettings {
	    catalogURL?: string;
	
	    static createFrom(source: any = {}) {
	        return new SoftwareSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.catalogURL = source["catalogURL"];
	    }
	}

}

export namespace tls {
	
	export class ConnectionState {
	    Version: number;
	    HandshakeComplete: boolean;
	    DidResume: boolean;
	    CipherSuite: number;
	    CurveID: number;
	    NegotiatedProtocol: string;
	    NegotiatedProtocolIsMutual: boolean;
	    ServerName: string;
	    PeerCertificates: x509.Certificate[];
	    VerifiedChains: x509.Certificate[][];
	    SignedCertificateTimestamps: number[][];
	    OCSPResponse: number[];
	    TLSUnique: number[];
	    ECHAccepted: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ConnectionState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Version = source["Version"];
	        this.HandshakeComplete = source["HandshakeComplete"];
	        this.DidResume = source["DidResume"];
	        this.CipherSuite = source["CipherSuite"];
	        this.CurveID = source["CurveID"];
	        this.NegotiatedProtocol = source["NegotiatedProtocol"];
	        this.NegotiatedProtocolIsMutual = source["NegotiatedProtocolIsMutual"];
	        this.ServerName = source["ServerName"];
	        this.PeerCertificates = this.convertValues(source["PeerCertificates"], x509.Certificate);
	        this.VerifiedChains = this.convertValues(source["VerifiedChains"], x509.Certificate);
	        this.SignedCertificateTimestamps = source["SignedCertificateTimestamps"];
	        this.OCSPResponse = source["OCSPResponse"];
	        this.TLSUnique = source["TLSUnique"];
	        this.ECHAccepted = source["ECHAccepted"];
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

export namespace unwxapp {
	
	export class AppidAPIKeyStatus {
	    Line: number;
	    Masked: string;
	    Status: string;
	    Detail: string;
	
	    static createFrom(source: any = {}) {
	        return new AppidAPIKeyStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Line = source["Line"];
	        this.Masked = source["Masked"];
	        this.Status = source["Status"];
	        this.Detail = source["Detail"];
	    }
	}
	export class VersionTaskStatus {
	    Number: string;
	    DecompileStatus: string;
	    MatchStatus: string;
	    Message: string;
	    WxapkgTags: string[];
	
	    static createFrom(source: any = {}) {
	        return new VersionTaskStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Number = source["Number"];
	        this.DecompileStatus = source["DecompileStatus"];
	        this.MatchStatus = source["MatchStatus"];
	        this.Message = source["Message"];
	        this.WxapkgTags = source["WxapkgTags"];
	    }
	}
	export class InfoToFront {
	    AppID: string;
	    UpdateDate: string;
	    Info?: model.MiniAppInfo;
	    Versions: VersionTaskStatus[];
	
	    static createFrom(source: any = {}) {
	        return new InfoToFront(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.AppID = source["AppID"];
	        this.UpdateDate = source["UpdateDate"];
	        this.Info = this.convertValues(source["Info"], model.MiniAppInfo);
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

export namespace url {
	
	export class Userinfo {
	
	
	    static createFrom(source: any = {}) {
	        return new Userinfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class URL {
	    Scheme: string;
	    Opaque: string;
	    // Go type: Userinfo
	    User?: any;
	    Host: string;
	    Path: string;
	    RawPath: string;
	    OmitHost: boolean;
	    ForceQuery: boolean;
	    RawQuery: string;
	    Fragment: string;
	    RawFragment: string;
	
	    static createFrom(source: any = {}) {
	        return new URL(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Scheme = source["Scheme"];
	        this.Opaque = source["Opaque"];
	        this.User = this.convertValues(source["User"], null);
	        this.Host = source["Host"];
	        this.Path = source["Path"];
	        this.RawPath = source["RawPath"];
	        this.OmitHost = source["OmitHost"];
	        this.ForceQuery = source["ForceQuery"];
	        this.RawQuery = source["RawQuery"];
	        this.Fragment = source["Fragment"];
	        this.RawFragment = source["RawFragment"];
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

export namespace webscan {
	
	export class PocConfig {
	    template_folders: string[];
	    auto_reload: boolean;
	    default_headers: string;
	    threads: number;
	    timeout: number;
	    rate_limit: number;
	    retries: number;
	
	    static createFrom(source: any = {}) {
	        return new PocConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.template_folders = source["template_folders"];
	        this.auto_reload = source["auto_reload"];
	        this.default_headers = source["default_headers"];
	        this.threads = source["threads"];
	        this.timeout = source["timeout"];
	        this.rate_limit = source["rate_limit"];
	        this.retries = source["retries"];
	    }
	}
	export class PocScanConfig {
	    targets: string[];
	    task_name: string;
	    append_poc: string;
	    tags: string[];
	    custom_tags: string[];
	    template_file: string[];
	    template_folders: string[];
	    custom_headers: string;
	    proxy: string;
	    skip_nuclei_without_tags: boolean;
	    thread_safe: boolean;
	    threads: number;
	    timeout: number;
	    rate_limit: number;
	    retries: number;
	
	    static createFrom(source: any = {}) {
	        return new PocScanConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.targets = source["targets"];
	        this.task_name = source["task_name"];
	        this.append_poc = source["append_poc"];
	        this.tags = source["tags"];
	        this.custom_tags = source["custom_tags"];
	        this.template_file = source["template_file"];
	        this.template_folders = source["template_folders"];
	        this.custom_headers = source["custom_headers"];
	        this.proxy = source["proxy"];
	        this.skip_nuclei_without_tags = source["skip_nuclei_without_tags"];
	        this.thread_safe = source["thread_safe"];
	        this.threads = source["threads"];
	        this.timeout = source["timeout"];
	        this.rate_limit = source["rate_limit"];
	        this.retries = source["retries"];
	    }
	}
	export class PocTaskRecord {
	    task_id: string;
	    name: string;
	    targets: string[];
	    vuln: number;
	    config: PocScanConfig;
	    created_at: string;
	    status: string;
	    progress: number;
	    completed: number;
	    total: number;
	    scanned_urls: string[];
	    results?: model.VulnerabilityInfo[];
	
	    static createFrom(source: any = {}) {
	        return new PocTaskRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.task_id = source["task_id"];
	        this.name = source["name"];
	        this.targets = source["targets"];
	        this.vuln = source["vuln"];
	        this.config = this.convertValues(source["config"], PocScanConfig);
	        this.created_at = source["created_at"];
	        this.status = source["status"];
	        this.progress = source["progress"];
	        this.completed = source["completed"];
	        this.total = source["total"];
	        this.scanned_urls = source["scanned_urls"];
	        this.results = this.convertValues(source["results"], model.VulnerabilityInfo);
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

export namespace x509 {
	
	export class PolicyMapping {
	    // Go type: OID
	    IssuerDomainPolicy: any;
	    // Go type: OID
	    SubjectDomainPolicy: any;
	
	    static createFrom(source: any = {}) {
	        return new PolicyMapping(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.IssuerDomainPolicy = this.convertValues(source["IssuerDomainPolicy"], null);
	        this.SubjectDomainPolicy = this.convertValues(source["SubjectDomainPolicy"], null);
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
	export class OID {
	
	
	    static createFrom(source: any = {}) {
	        return new OID(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class Certificate {
	    Raw: number[];
	    RawTBSCertificate: number[];
	    RawSubjectPublicKeyInfo: number[];
	    RawSubject: number[];
	    RawIssuer: number[];
	    Signature: number[];
	    SignatureAlgorithm: number;
	    PublicKeyAlgorithm: number;
	    PublicKey: any;
	    Version: number;
	    // Go type: big
	    SerialNumber?: any;
	    Issuer: pkix.Name;
	    Subject: pkix.Name;
	    // Go type: time
	    NotBefore: any;
	    // Go type: time
	    NotAfter: any;
	    KeyUsage: number;
	    Extensions: pkix.Extension[];
	    ExtraExtensions: pkix.Extension[];
	    UnhandledCriticalExtensions: number[][];
	    ExtKeyUsage: number[];
	    UnknownExtKeyUsage: number[][];
	    BasicConstraintsValid: boolean;
	    IsCA: boolean;
	    MaxPathLen: number;
	    MaxPathLenZero: boolean;
	    SubjectKeyId: number[];
	    AuthorityKeyId: number[];
	    OCSPServer: string[];
	    IssuingCertificateURL: string[];
	    DNSNames: string[];
	    EmailAddresses: string[];
	    IPAddresses: number[][];
	    URIs: url.URL[];
	    PermittedDNSDomainsCritical: boolean;
	    PermittedDNSDomains: string[];
	    ExcludedDNSDomains: string[];
	    PermittedIPRanges: net.IPNet[];
	    ExcludedIPRanges: net.IPNet[];
	    PermittedEmailAddresses: string[];
	    ExcludedEmailAddresses: string[];
	    PermittedURIDomains: string[];
	    ExcludedURIDomains: string[];
	    CRLDistributionPoints: string[];
	    PolicyIdentifiers: number[][];
	    Policies: OID[];
	    InhibitAnyPolicy: number;
	    InhibitAnyPolicyZero: boolean;
	    InhibitPolicyMapping: number;
	    InhibitPolicyMappingZero: boolean;
	    RequireExplicitPolicy: number;
	    RequireExplicitPolicyZero: boolean;
	    PolicyMappings: PolicyMapping[];
	
	    static createFrom(source: any = {}) {
	        return new Certificate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Raw = source["Raw"];
	        this.RawTBSCertificate = source["RawTBSCertificate"];
	        this.RawSubjectPublicKeyInfo = source["RawSubjectPublicKeyInfo"];
	        this.RawSubject = source["RawSubject"];
	        this.RawIssuer = source["RawIssuer"];
	        this.Signature = source["Signature"];
	        this.SignatureAlgorithm = source["SignatureAlgorithm"];
	        this.PublicKeyAlgorithm = source["PublicKeyAlgorithm"];
	        this.PublicKey = source["PublicKey"];
	        this.Version = source["Version"];
	        this.SerialNumber = this.convertValues(source["SerialNumber"], null);
	        this.Issuer = this.convertValues(source["Issuer"], pkix.Name);
	        this.Subject = this.convertValues(source["Subject"], pkix.Name);
	        this.NotBefore = this.convertValues(source["NotBefore"], null);
	        this.NotAfter = this.convertValues(source["NotAfter"], null);
	        this.KeyUsage = source["KeyUsage"];
	        this.Extensions = this.convertValues(source["Extensions"], pkix.Extension);
	        this.ExtraExtensions = this.convertValues(source["ExtraExtensions"], pkix.Extension);
	        this.UnhandledCriticalExtensions = source["UnhandledCriticalExtensions"];
	        this.ExtKeyUsage = source["ExtKeyUsage"];
	        this.UnknownExtKeyUsage = source["UnknownExtKeyUsage"];
	        this.BasicConstraintsValid = source["BasicConstraintsValid"];
	        this.IsCA = source["IsCA"];
	        this.MaxPathLen = source["MaxPathLen"];
	        this.MaxPathLenZero = source["MaxPathLenZero"];
	        this.SubjectKeyId = source["SubjectKeyId"];
	        this.AuthorityKeyId = source["AuthorityKeyId"];
	        this.OCSPServer = source["OCSPServer"];
	        this.IssuingCertificateURL = source["IssuingCertificateURL"];
	        this.DNSNames = source["DNSNames"];
	        this.EmailAddresses = source["EmailAddresses"];
	        this.IPAddresses = source["IPAddresses"];
	        this.URIs = this.convertValues(source["URIs"], url.URL);
	        this.PermittedDNSDomainsCritical = source["PermittedDNSDomainsCritical"];
	        this.PermittedDNSDomains = source["PermittedDNSDomains"];
	        this.ExcludedDNSDomains = source["ExcludedDNSDomains"];
	        this.PermittedIPRanges = this.convertValues(source["PermittedIPRanges"], net.IPNet);
	        this.ExcludedIPRanges = this.convertValues(source["ExcludedIPRanges"], net.IPNet);
	        this.PermittedEmailAddresses = source["PermittedEmailAddresses"];
	        this.ExcludedEmailAddresses = source["ExcludedEmailAddresses"];
	        this.PermittedURIDomains = source["PermittedURIDomains"];
	        this.ExcludedURIDomains = source["ExcludedURIDomains"];
	        this.CRLDistributionPoints = source["CRLDistributionPoints"];
	        this.PolicyIdentifiers = source["PolicyIdentifiers"];
	        this.Policies = this.convertValues(source["Policies"], OID);
	        this.InhibitAnyPolicy = source["InhibitAnyPolicy"];
	        this.InhibitAnyPolicyZero = source["InhibitAnyPolicyZero"];
	        this.InhibitPolicyMapping = source["InhibitPolicyMapping"];
	        this.InhibitPolicyMappingZero = source["InhibitPolicyMappingZero"];
	        this.RequireExplicitPolicy = source["RequireExplicitPolicy"];
	        this.RequireExplicitPolicyZero = source["RequireExplicitPolicyZero"];
	        this.PolicyMappings = this.convertValues(source["PolicyMappings"], PolicyMapping);
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

