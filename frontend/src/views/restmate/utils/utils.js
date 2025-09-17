export const ENVIRONMENT_REGEX = /\{\{([\w.-]+)\}\}/g;
export const extractEnv = (x) => {
  return x.replace(/\{\{([\w.-]+)\}\}/, "$1");
};

export const cleanUpRequest = (t) => {
  if (!t) {
    return t;
  }
  t.headers = t.headers && t.headers.filter((h) => h.key !== "" && h.active === true);
  t.params = t.params && t.params.filter((h) => h.key !== "" && h.active === true);
  t.body.formData = t.body.formData && t.body.formData.filter((h) => h.key !== "" && h.active === true);
  if (t.body?.bodyType === "json") {
    t.body.formData = [];
  }
  if (t.body?.bodyType === "formdata") {
    t.body.bodyRaw = "";
  }
  if (t.body?.bodyType === "none") {
    t.body.bodyRaw = "";
    t.body.formData = [];
  }
  return t;
};

export const hotkeys_json = [
  //comps/misc/SideBar.jsx
  {
    name: "显示/隐藏侧边栏",
    key: ["Ctrl", "B"],
  },
  {
    name: "创建新的集合",
    key: ["Ctrl", "N"],
  },
  //comps/TabsRoot.jsx
  {
    name: "创建新标签",
    key: ["Ctrl", "T"],
  },
  {
    name: "切换下一个标签",
    key: ["Ctrl", "Right"],
  },
  {
    name: "切换上一个标签",
    key: ["Ctrl", "Left"],
  },
  //comps/ReqHead.jsx
  {
    name: "保存请求",
    key: ["Ctrl", "S"],
  },
  {
    name: "调用请求",
    key: ["Ctrl", "Enter"],
  },
  {
    name: "关闭标签",
    key: ["Ctrl", "W"],
  },
];
export const colors = [
  {
    name: "light",
    palettes: ["#ffffff", "#1a1b27", "#c53b53"],
  },
  {
    name: "dark",
    palettes: ["#212121", "#262626", "#6366fb"],
  },
];
