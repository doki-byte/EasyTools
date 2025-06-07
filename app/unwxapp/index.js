/*!
 * unveilr v2.0.1
 * (c) 2023 r3x5ur
 * Released under the GPL-3.0 License.
 */

"use strict";
var e = require("./traverse-252284fd.js"),
  t = require("threads"),
  r = require("os"),
  s = require("commander");
require("./parser-928e23b1.js"),
  require("buffer"),
  require("module"),
  require("vm"),
  require("fs"),
  require("path"),
  require("events"),
  require("util"),
  require("async_hooks"),
  require("worker_threads"),
  require("observable-fns"),
  require("chalk"),
  require("winston"),
  require("fs/promises"),
  require("crypto-js"),
  require("prettier"),
  require("css-tree"),
  require("@babel/core");
class a extends e.BaseError {}
class i extends e.BaseLogger {
  constructor(t) {
    super(), (this.pathCtrl = e.PathController.make(t));
  }
  get extractable() {
    return this.pathCtrl.isFile && this.pathCtrl.suffixWithout === this.suffix;
  }
  extract() {
    this.extractable ||
      a.throw(`File ${this.pathCtrl.logpath} cannot be extracted!`);
  }
}
class o extends e.BaseError {}
class p extends e.BaseLogger {
  constructor(t) {
    if (e.isWorkerRuntime()) throw Error("Saver cannot run on Worker!");
    super(),
      (this.saveList = []),
      (this.saveDirectory = t),
      (this.saveCtrl = e.getSaveController());
  }
  set saveDirectory(t) {
    this.baseDir = e.PathController.make(t);
  }
  get saveDirectory() {
    return this.baseDir;
  }
  add(t, r, s) {
    const a = e.PathController.make(t);
    return (
      a.isAbs
        ? this.saveCtrl.set(a.path, r)
        : s
        ? this.saveCtrl.set(a.abspath, r)
        : this.saveList.push({ path: t, buffer: r }),
      this
    );
  }
  merge() {
    return (
      this.saveList.forEach((t) => {
        const { path: r, buffer: s } = t,
          a = e.PathController.make(r);
        let i = a;
        a.isAbs ||
          (this.baseDir || o.throw("BaseDir is not a directory!"),
          (i = this.baseDir.join(a.path))),
          this.saveCtrl.set(i.abspath, s);
      }),
      this
    );
  }
}
class n extends e.BaseError {}
class h extends e.BaseLogger {
  constructor(t) {
    super(),
      (this.pathCtrl = e.PathController.make(t)),
      (this.saver = new p(this.pathCtrl.dirname));
  }
  get decipherable() {
    return this.pathCtrl.isFile && this.pathCtrl.suffixWithout === this.suffix;
  }
  decrypt() {
    this.decipherable ||
      n.throw(`File ${this.pathCtrl.logpath} cannot be decrypted!`);
  }
}
function c(t) {
  const r = e.isProduciblePath(t) ? e.PathController.make(t).readSync() : t;
  return 190 === r.readUInt8(0) && 237 === r.readUInt8(13);
}
function l(t) {
  if (t.every((e) => e.startsWith("WA"))) return e.WxapkgType.FRAMEWORK;
  if (t.includes(e.WxapkgKeyFile.PAGE_FRAME_HTML))
    return t.includes(e.WxapkgKeyFile.COMMON_APP)
      ? e.WxapkgType.APP_V4
      : e.WxapkgType.APP_V1;
  if (t.includes(e.WxapkgKeyFile.COMMON_APP))
    return t.includes(e.WxapkgKeyFile.APP_WXSS)
      ? e.WxapkgType.APP_V3
      : e.WxapkgType.APP_SUBPACKAGE_V2;
  if (t.includes(e.WxapkgKeyFile.PAGE_FRAME))
    return t.includes(e.WxapkgKeyFile.APP_WXSS)
      ? e.WxapkgType.APP_V2
      : e.WxapkgType.APP_SUBPACKAGE_V1;
  if (t.includes(e.WxapkgKeyFile.GAME))
    return t.includes(e.WxapkgKeyFile.APP_CONFIG)
      ? e.WxapkgType.GAME
      : e.WxapkgType.GAME_SUBPACKAGE;
  if (t.includes(e.WxapkgKeyFile.PLUGIN_JSON)) {
    if (t.includes(e.WxapkgKeyFile.APPSERVICE))
      return e.WxapkgType.APP_PLUGIN_V1;
    if (t.includes(e.WxapkgKeyFile.PLUGIN)) return e.WxapkgType.GAME_PLUGIN;
  }
  return null;
}
class g extends h {
  constructor(t, r, s) {
    let a;
    (a =
      1 === arguments.length
        ? e.isProduciblePath(t)
          ? { path: t }
          : t
        : { path: t, wxAppId: r, target: 3 === arguments.length ? s : void 0 }),
      super(a.path),
      (this.suffix = e.PackageSuffix.WXAPKG),
      (this.wxAppId = a.wxAppId),
      (this.salt = a.salt || "saltiest"),
      (this.iv = a.iv || "the iv: 16 bytes"),
      (this.target = a.target ? e.PathController.make(a.target) : void 0),
      this._calcWxAppId();
  }
  get result() {
    return this.decryptedBuffer;
  }
  _calcWxAppId() {
    if (this.wxAppId) return;
    const t = this.pathCtrl.abspath.match(/wx[a-z\d]{16}/g);
    if (!t) throw new n("wxAppId must be required!");
    (this.wxAppId = t[0]),
      this.logger.info(
        `From ${this.pathCtrl.logpath} detected wxAppId: ${e.info(
          this.wxAppId
        )}`
      );
  }
  checkWxAppId() {
    if (!this.wxAppId || !/^wx[a-z\d]{16}$/.test(this.wxAppId))
      throw new n(`wxAppId ${this.wxAppId || ""} must be a valid wxAppId`);
  }
  decrypt(e) {
    super.decrypt(), (e = e || this.pathCtrl.readSync()), this._decrypt(e);
  }
  _decrypt(t) {
    try {
      this.checkWxAppId();
      const r = this.wxAppId,
        s = t.subarray(6, 1030),
        a = t.subarray(1030),
        i = e.decryptBuffer(s, r, this.salt, this.iv),
        o = r.length < 2 ? 102 : r.charCodeAt(r.length - 2),
        p = Buffer.from(a.map((e) => e ^ o));
      (this.decryptedBuffer = Buffer.concat([i.subarray(0, 1023), p])),
        c(this.decryptedBuffer) ||
          n.throw("Please check if wxAppId is correct"),
        this.logger.debug("Decryption successful!");
    } catch (e) {
      throw new n("Decryption failed! " + e.message);
    }
  }
  async save(e) {
    this.saver.add(e || this.target, this.result);
  }
  static decryptResult(t, r) {
    const s = e.isProduciblePath(t) ? new g(t, r) : new g(t);
    return s.decrypt(), s.result;
  }
}
class u extends i {
  constructor(t) {
    if (e.isProduciblePath(t)) super(t);
    else {
      const { path: e, saveDir: r, wxAppId: s } = t;
      super(e), this.setSaver(r), (this.wxAppId = s);
    }
    (this.isExtracted = !1), (this.suffix = e.PackageSuffix.WXAPKG);
  }
  get extracted() {
    return this.isExtracted || a.throw("Need to extract first"), !0;
  }
  get type() {
    return (
      this.wxapkgType ||
        a.throw("WxapkgType not available, No extract or unsupported packages"),
      this.wxapkgType
    );
  }
  get isMainPackage() {
    return (
      this.type === e.WxapkgType.APP_V1 ||
      this.type === e.WxapkgType.APP_V2 ||
      this.type === e.WxapkgType.APP_V3 ||
      this.type === e.WxapkgType.APP_V4 ||
      this.type === e.WxapkgType.GAME
    );
  }
  get isSubpackage() {
    return (
      this.type === e.WxapkgType.APP_SUBPACKAGE_V1 ||
      this.type === e.WxapkgType.APP_SUBPACKAGE_V2 ||
      this.type === e.WxapkgType.GAME_SUBPACKAGE
    );
  }
  get isAppPlugin() {
    return this.type === e.WxapkgType.APP_PLUGIN_V1;
  }
  get isGamePlugin() {
    return this.type === e.WxapkgType.GAME_PLUGIN;
  }
  get isPlugin() {
    return this.isAppPlugin || this.isGamePlugin;
  }
  get isFramework() {
    return this.type === e.WxapkgType.FRAMEWORK;
  }
  setSaver(e) {
    (e = e || this.pathCtrl.whitout()),
      this.saver ? (this.saver.saveDirectory = e) : (this.saver = new p(e));
  }
  setWxAppId(e) {
    this.wxAppId = e;
  }
  save() {
    this.saver.merge();
  }
  get saveDirectory() {
    return this.saver.saveDirectory;
  }
  get sourceDir() {
    return this.saveDirectory.join(this.sourcePath);
  }
  getFileHeader(e) {
    c(e) || a.throw(`File ${this.pathCtrl.logpath} is an invalid package!`);
    const t = e.readUInt32BE(1);
    return (
      t && this.logger.warn("UnknownInfo: ", t),
      { infoLength: e.readUInt32BE(5), dataLength: e.readUInt32BE(9) }
    );
  }
  getFileByRaw(e) {
    const t = e.readUInt32BE(0);
    this.logger.debug(`Read file count ${t}`);
    let r = 4;
    return Array(t)
      .fill(0)
      .map(() => {
        const t = e.readUInt32BE(r);
        r += 4;
        const s = e.toString("utf8", r, r + t);
        r += t;
        const a = e.readUInt32BE(r);
        r += 4;
        const i = a + e.readUInt32BE(r);
        return (r += 4), { name: s, start: a, end: i };
      });
  }
  extractInner(t) {
    if ("56314d4d5758" === t.subarray(0, 6).toString("hex")) {
      this.logger.debug(
        `File ${this.pathCtrl.logpath} encrypted, Starting decrypt`
      );
      const e = g.decryptResult(this.pathCtrl, this.wxAppId);
      return this.extractInner(e);
    }
    this.logger.debug(`Starting extract ${this.pathCtrl.logpath}`);
    const { dataLength: r, infoLength: s } = this.getFileHeader(
      t.subarray(0, 14)
    );
    this.logger.debug(`Header info length ${s}`),
      this.logger.debug(`Header data length ${r}`);
    const i = this.getFileByRaw(t.subarray(14, s + 14));
    this.logger.debug("Starting save extracted files");
    const o = i.map((r) => {
        const { name: s, start: a, end: i } = r,
          o = s.startsWith("/") ? s.slice(1) : s;
        this.saver.add(o, t.subarray(a, i));
        return { path: o, basename: e.PathController.make(o).basename };
      }),
      p = (function (t) {
        if (e.isProduciblePath(t)) {
          const r = e.PathController.make(t);
          if (!r.isDirectory) throw Error(`Path ${t} is not a directory`);
          return r.readdir().then((e) => l(e));
        }
        return l(t);
      })(o.map(({ basename: e }) => e));
    if (
      (p || this.logger.warn("Parsed packages are not supported"),
      this.logger.info(
        `The package ${this.pathCtrl.logpath} type is: [${e.info(p)}]`
      ),
      (this.wxapkgType = p),
      p === e.WxapkgType.FRAMEWORK)
    )
      return (
        this.logger.warn("Running the framework does not require unpacking"),
        void (this.isExtracted = !0)
      );
    const n = this.isPlugin;
    let h = null;
    o.forEach((t) => {
      const { basename: r } = t;
      if (
        !(
          r === e.WxapkgKeyFile.APP_SERVICE ||
          r === e.WxapkgKeyFile.APPSERVICE ||
          r === e.WxapkgKeyFile.GAME ||
          (n && r === e.WxapkgKeyFile.PLUGIN_JSON)
        )
      )
        return;
      if (!h) return (h = t);
      const s = t.path.split("/").length,
        a = h.path.split("/").length;
      h = s < a ? t : h;
    }),
      h || a.throw(`File ${this.pathCtrl.logpath} source directory not found`),
      (this.sourcePath = e.PathController.make(h.path || ".").dirname),
      (this.isExtracted = !0);
  }
  extract(t) {
    super.extract();
    const r = this.pathCtrl.readSync();
    if (
      !(function (e) {
        const t = "WAPkgEncryptedTagForMac";
        return e.subarray(-t.length).toString() !== t;
      })(r)
    ) {
      const t = e.link(
        e.info("https://github.com/TinyNiko/mac_wxapkg_decrypt")
      );
      a.throw(
        `Package ${this.pathCtrl.logpath} is an encrypted package for Mac\n    please use ${t} to decrypt it before using it`
      );
    }
    this.extractInner(r), t && this.save();
  }
}
class d extends e.BaseError {}
class x {
  constructor(t, s) {
    (this.tasks = []), (this.results = []);
    var a;
    (this.workerPath =
      "string" == typeof t
        ? t
        : ((a = t.id), e.PathController.make(__dirname).relative(a).unixpath)),
      (this.poolSize = s || r.cpus().length);
  }
  async start(e, r) {
    let s,
      a = r;
    "function" == typeof e ? (s = e) : (a = Boolean(e)),
      this.tasks.length &&
        ((this.pool = t.Pool(
          () => t.spawn(new t.Worker(this.workerPath)),
          this.poolSize
        )),
        (this.results = this.tasks.map((e) => this.pool.queue(e))),
        await this.pool.completed(),
        (this.tasks = []),
        s && (await this.forEach(s)),
        a && (await this.terminate()));
  }
  addTask(...e) {
    e.forEach((e) => this.tasks.push(e));
  }
  async terminate() {
    this.pool ||
      d.throw("The worker pool has not started yet, please start it first"),
      await this.pool.terminate(),
      delete this.pool;
  }
  async forEach(e) {
    await Promise.all(
      this.results.map((t, r) => t.then((t) => e(t, r, this.results)))
    ),
      (this.results = []);
  }
}
class P extends e.BaseError {}
class f extends e.BaseLogger {
  get isParserV1() {
    const t = this.extractor.type;
    return (
      t === e.WxapkgType.APP_V1 ||
      t === e.WxapkgType.APP_V2 ||
      t === e.WxapkgType.APP_SUBPACKAGE_V1
    );
  }
  get isParserV3() {
    const t = this.extractor.type;
    return (
      t === e.WxapkgType.APP_V3 ||
      t === e.WxapkgType.APP_V4 ||
      t === e.WxapkgType.APP_SUBPACKAGE_V2 ||
      t === e.WxapkgType.APP_PLUGIN_V1
    );
  }
  get isMainPackage() {
    return this.extractor.isMainPackage;
  }
  get isSubpackage() {
    return this.extractor.isSubpackage;
  }
  get isAppPlugin() {
    return this.extractor.isAppPlugin;
  }
  get path() {
    return this.pathCtrl;
  }
  get sourceDir() {
    return this.extractor.sourceDir;
  }
  get saveDir() {
    return this.saver.saveDirectory;
  }
  set saveDir(e) {
    (e = e || this.path.whitout()),
      this.extractor.setSaver(e),
      (this.saver.saveDirectory = e);
  }
  constructor(t) {
    super(),
      (this.pathCtrl = e.PathController.make(t)),
      (this.extractor = new u({ path: this.path })),
      (this.saver = new p(this.extractor.saveDirectory)),
      (this.parsers = Object.create(null)),
      (this.traverseList = []);
  }
  setExtractorWxAppId(e) {
    this.extractor.setWxAppId(e);
  }
  extract() {
    this.extractor.extract();
  }
  extractorSave() {
    this.extractor.save();
  }
  async makeParserTraverse() {
    if (this.extractor.isFramework) return [];
    this.initParsers(), await this.initTraverseList();
    const e = this.traverseList.filter((e) => e.source);
    return (
      e.length ||
        this.logger.warn(`File ${this.path.logpath} no data to parse`),
      e.map((e) => Object.assign({ decompiler: this }, e))
    );
  }
  initParsers() {
    if (this.extractor.extracted)
      switch (
        ((this.parsers.ScriptParser = new e.ScriptParser(this.saver)),
        this.extractor.type)
      ) {
        case e.WxapkgType.APP_V1:
        case e.WxapkgType.APP_V2:
        case e.WxapkgType.APP_V3:
        case e.WxapkgType.APP_V4:
        case e.WxapkgType.APP_SUBPACKAGE_V1:
        case e.WxapkgType.APP_SUBPACKAGE_V2:
        case e.WxapkgType.APP_PLUGIN_V1:
          {
            const t = new e.WxssParser(this.saver);
            (this.parsers.WxssParser = t),
              (this.parsers.WxssParserCommon = t),
              (this.parsers.WxssParserCommon2 = t),
              this.isSubpackage ||
                (this.parsers.AppConfigService = new e.AppConfigParser(
                  this.saver
                )),
              this.isParserV1
                ? (this.parsers.WxmlParserV1 = new e.WxmlParser(this.saver))
                : this.isParserV3 &&
                  (this.parsers.WxmlParserV3 = new e.WxmlParser(this.saver));
          }
          break;
        case e.WxapkgType.GAME:
          this.parsers.AppConfigService = new e.AppConfigParser(this.saver);
      }
  }
  async initTraverseList() {
    if (!this.extractor.extracted) return;
    const t = this.sourceDir,
      r = (r) => {
        const s = t.join(r);
        return e.saveAble2String(e.getSaveController().get(s.abspath));
      },
      s = async (s) => {
        const {
            serviceSource: a = r(e.WxapkgKeyFile.APP_SERVICE),
            viewSource: i = r(e.WxapkgKeyFile.APP_WXSS),
            appConfigSource: o,
            setAppConfig: p = !0,
          } = s || {},
          n = (function (e) {
            if (!e) return null;
            const t = e.match(/\/\*(v\S+?)\*\//);
            return Array.isArray(t) && t[1] ? t[1] : null;
          })(i);
        if (
          (n &&
            this.logger.info(
              `The package ${this.pathCtrl.logpath} wcc version is: [${e.info(
                n
              )}]`
            ),
          p)
        ) {
          const t = o || r(e.WxapkgKeyFile.APP_CONFIG),
            s = this.parsers.AppConfigService;
          s.setSources(t),
            s.setServiceSource(a),
            this.traverseList.push({
              source: a,
              visitors: ["AppConfigService", "ScriptParser"],
            });
        } else
          this.traverseList.push({ source: a, visitors: ["ScriptParser"] });
        const h = [i, e.WxssParser.getHTMLStyleSource(t)].join(";\n");
        if (
          (this.traverseList.push({
            source: h,
            visitors: ["WxssParser", "WxssParserCommon", "WxssParserCommon2"],
          }),
          this.isParserV1)
        ) {
          this.traverseList.push({ source: i, visitors: ["WxmlParserV1"] });
          this.parsers.WxmlParserV1.setSource(i);
        } else
          this.isParserV3 &&
            this.traverseList.push({ source: i, visitors: ["WxmlParserV3"] });
      },
      a = async (e) => {
        this.traverseList.push({ source: r(e), visitors: ["ScriptParser"] });
      };
    switch (this.extractor.type) {
      case e.WxapkgType.APP_V1:
      case e.WxapkgType.APP_V4:
        await s({
          viewSource: e.matchScripts(r(e.WxapkgKeyFile.PAGE_FRAME_HTML)),
        });
        break;
      case e.WxapkgType.APP_V2:
      case e.WxapkgType.APP_V3:
        await s();
        break;
      case e.WxapkgType.APP_SUBPACKAGE_V1:
      case e.WxapkgType.APP_SUBPACKAGE_V2:
        await s({
          viewSource: r(e.WxapkgKeyFile.PAGE_FRAME),
          setAppConfig: !1,
        });
        break;
      case e.WxapkgType.APP_PLUGIN_V1:
        await s({
          viewSource: r(e.WxapkgKeyFile.PAGEFRAME),
          serviceSource: r(e.WxapkgKeyFile.APPSERVICE),
          setAppConfig: !1,
        });
        break;
      case e.WxapkgType.GAME:
        {
          const t = r(e.WxapkgKeyFile.APP_CONFIG),
            s = [r(e.WxapkgKeyFile.GAME), r(e.WxapkgKeyFile.SUBCONTEXT)]
              .filter(Boolean)
              .join(";\n"),
            i = this.parsers.AppConfigService;
          i.setSources(t),
            i.setIsGame(!0),
            this.traverseList.push({
              source: s,
              visitors: ["AppConfigService"],
            }),
            await a(e.WxapkgKeyFile.GAME);
        }
        break;
      case e.WxapkgType.GAME_SUBPACKAGE:
        await a(e.WxapkgKeyFile.GAME);
        break;
      case e.WxapkgType.GAME_PLUGIN:
        await a(e.WxapkgKeyFile.PLUGIN);
    }
  }
  save() {
    this.extractor.isFramework ||
      (this.saver.merge(), e.getConfig("WXClearDecompile") && this.cleanup());
  }
  cleanup() {
    const t = this.sourceDir;
    this.logger.debug(`Start cleaning ${t.logpath}`);
    const r = e.getSaveController();
    [
      ".appservice.js",
      "appservice.js",
      "app-config.json",
      "app-service.js",
      "app-wxss.js",
      "appservice.app.js",
      "common.app.js",
      "page-frame.js",
      "page-frame.html",
      "pageframe.js",
      "webview.app.js",
      "subContext.js",
      "plugin.js",
    ].forEach((e) => r.delete(t.join(e).abspath));
  }
}
class y extends e.BaseLogger {
  constructor(t) {
    super();
    const r = Object.create(null);
    Array.isArray(t) ? (r.wxapkgList = t) : Object.assign(r, t);
    const { wxapkgList: s, mainSaveDir: a, wxAppId: i } = r;
    a && (this.mainSaveDir = e.PathController.make(a)),
      s.length || P.throw("No wxapkg available"),
      (this.decompilers = s.map((e) => {
        const t = new f(e);
        return i && t.setExtractorWxAppId(i), t;
      })),
      (this.parsersPromise = []);
  }
  extractorSave() {
    this.clearOldResult(), this.decompilers.forEach((e) => e.extractorSave());
  }
  extract(t) {
    if ((this.decompilers.forEach((e) => e.extract()), t))
      return this.extractorSave();
    let r;
    if (
      (this.decompilers.forEach((e) => {
        e.isMainPackage &&
          (r && P.throw("There can only be one main package"), (r = e));
      }),
      !r)
    )
      return this.extractorSave();
    if (this.mainSaveDir) {
      const t = this.mainSaveDir;
      (r.saveDir = t),
        t.isDirectory &&
          t.deepListDir(1).length &&
          (e.getConfig("WxClearOutput") ||
            P.throw(
              `This output path is not empty, please use ${e.info(
                "--clear-output"
              )} force cleanup`
            ),
          r.saveDir.rmrfSync());
    }
    this.decompilers.forEach((e) => {
      e !== r && (e.isAppPlugin || (e.saveDir = r.saveDir));
    }),
      this.extractorSave();
  }
  clearOldResult() {
    e.getConfig("WXClearSave") &&
      Array.from(new Set(this.decompilers.map((e) => e.saveDir))).forEach(
        (e) => {
          e.rmrfSync(), this.logger.debug(`Cleaned old result ${e.logpath}`);
        }
      );
  }
  async invokeTraverseWorker() {
    const t = (e, t, r) => {
        const s = new Set();
        t.forEach((t) => {
          if ("WxmlParserV1" === t) {
            return e.parsers.WxmlParserV1.parseV1();
          }
          if ("AppConfigService" === t) {
            const t = e.parsers.AppConfigService;
            if (t.isGame) return t.parseGame();
          }
          const a = e.parsers[t];
          r.setVisitor(t),
            !s.has(a) && this.parsersPromise.push(a.parse(r.observable())),
            s.add(a);
        });
      },
      r = await Promise.all(
        this.decompilers.map((e) => e.makeParserTraverse())
      ),
      s = [];
    r.forEach((e) => {
      e && s.push(...e);
    });
    const a = s.filter(Boolean);
    if (a.length)
      if (1 === a.length) {
        const { decompiler: r, visitors: s, source: i } = a[0],
          o = e.createExposed();
        t(r, s, o), await o.startTraverse(i);
      } else {
        const r = new x(e.traverseModule);
        a.forEach((s) => {
          const { decompiler: a, visitors: i, source: o } = s;
          r.addTask(
            (r) => (
              r.initWorker(e.getInnerConfig()), t(a, i, r), r.startTraverse(o)
            )
          );
        }),
          await r.start(!0);
      }
    else this.logger.warn("No task to traverse");
  }
  async saveFiles() {
    await Promise.all(this.parsersPromise),
      this.decompilers.forEach((e) => e.save()),
      await e.flashDisk();
  }
  async exploit() {
    const t = e.getConfig("WXParse");
    this.extract(!t),
      t && (await this.invokeTraverseWorker()),
      await this.saveFiles();
  }
}
const k = {
    writeOut(t) {
      process.stdout.write(e.green(t));
    },
    writeErr(t) {
      process.stdout.write(e.yellow(t));
    },
    outputError(t, r) {
      r(e.red(t));
    },
  },
  A = new s.Command("wx"),
  v = new s.Option(
    "-p, --no-parse",
    "Only extract files, do not parse"
  ).implies({ clearDecompile: !1 }),
  w = new s.Option("-d, --depth <depth>", "Set read-depth")
    .argParser((e) => {
      const t = parseInt(e);
      if (isNaN(t))
        throw new s.CommanderError(1, "depth.error", "Invalid depth");
      return t;
    })
    .default(1);
A.option(
  "-i, --appid <appid>",
  "Set wxAppId, not provided will try to fetch from path"
)
  .option("-f, --format", "Enable format code")
  .option("--no-clear-decompile", "Retain decompiling residual files")
  .option("--no-clear-save", "The path to be saved will not be cleared")
  .addOption(v)
  .addOption(w)
  .option(
    "-o, --output <path>",
    "Set output path, default: main package whit out"
  )
  .option("--clear-output", "Empty the specified output folder")
  .description("Decompile the WeChat applet")
  .addArgument(
    new s.Argument(
      "<packages...>",
      "Set package path, could be a file, directory or multiple files"
    )
  )
  .showHelpAfterError()
  .configureOutput(k);
var m = A;
var W = "unveilr",
  C = "2.0.1";
function S(t) {
  return t || "ts" === e.PathController.make(process.argv[1]).suffixWithout
    ? {
        global: { logLevel: "info" },
        wx: {
          format: !0,
          clearDecompile: !0,
          clearSave: !0,
          parse: !0,
          depth: 1,
          packages: ["files"],
        },
      }
    : (function (e, t, r) {
        const a = r || process.argv,
          i = new s.Option("-l, --log-level <level>", "Set log level")
            .choices(["debug", "info", "warn", "error"])
            .default("info"),
          o = new s.Command(t);
        return (
          o
            .usage("[wx] [options]")
            .version(e, "-v, --version")
            .addOption(i)
            .addCommand(m, { isDefault: !0 })
            .addHelpText(
              "after",
              `\nExample:\n  $ ${t} /path/to/wxapkg/dir/\n  $ ${t} 1.wxapkg 2.wxapkg 3.wxapkg ...\n  $ ${t} wx /path/to/wxapkg/dir/           Specify wx subcommand\n  $ ${t} wx 1.wxapkg 2.wxapkg 3.wxapkg ... Specify wx subcommand\n  $ ${t} wx -h                             Show wx help info\n`
            )
            .showHelpAfterError()
            .configureOutput(k)
            .parse(a),
          a.length
            ? { global: o.opts(), wx: { ...m.opts(), packages: m.args } }
            : o.help({ error: !0 })
        );
      })(C, W);
}
async function E() {
  var t;
  e.initializeConfig(S()),
    process.stdout.write("[2J"),
    process.stdout.write("[1;1H"),
    t && process.stdout.write("[3J"),
    e.registerGlobalException(),
    e.SaveController.setIsClean(e.getConfig("WXClearDecompile")),
    e.SaveController.setIsReFormat(e.getConfig("WXReformat"));
  const r = e.getConfig("WXPackages");
  function s(t) {
    return t.isFile && t.suffixWithout === e.PackageSuffix.WXAPKG;
  }
  const a = e.getConfig("WXDepth"),
    i = [];
  for (const t of r) {
    const r = e.PathController.make(t);
    if (r.isDirectory) {
      const t = r
        .deepListDir(a)
        .map((t) => e.PathController.make(t))
        .filter((e) => s(e));
      i.push(...t);
    } else i.push(r);
  }
  return new y({
    wxapkgList: i,
    wxAppId: e.getConfig("WXAppId"),
    mainSaveDir: e.getConfig("WXOutput"),
  }).exploit();
}
E().then(), (exports.main = E);
