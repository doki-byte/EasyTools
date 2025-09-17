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
	
	export class GitHubRelease {
	    tag_name: string;
	    html_url: string;
	
	    static createFrom(source: any = {}) {
	        return new GitHubRelease(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tag_name = source["tag_name"];
	        this.html_url = source["html_url"];
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

export namespace proxy {
	
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

export namespace struct {} {
	
	export class  {
	
	
	    static createFrom(source: any = {}) {
	        return new (source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}

}

