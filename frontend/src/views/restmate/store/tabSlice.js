import { nanoid } from "nanoid";
import { GetRequest, InvokeRequest } from "../../../../wailsjs/go/restmate/RestMate";
import { ChoseFile } from "../../../../wailsjs/go/restmate/RestMate";
import { cleanUpRequest } from "../utils/utils";

export function tabSchema(data = {}) {
  const defaults = {
    id: nanoid(),
    name: "Untitled",
    url: "",
    method: "get",
    headers: [{ id: nanoid(), key: "", value: "", active: true }],
    params: [{ id: nanoid(), key: "", value: "", active: true }],
    body: {
      bodyType: "json",
      bodyRaw: "",
      formData: [
        {
          id: nanoid(),
          key: "",
          value: "",
          files: [],
          type: "file",
          active: true,
        },
      ],
    },
    coll_id: null,
    response: null,
  };
  return { ...defaults, ...data };
}
export const createTabsSlice = (set, get) => ({
  tabs: [tabSchema()],

  invokeLoading: false,
  invokeReq: async (id) => {
    set({ invokeLoading: true });
    let tab = get().tabs.find((t) => t.id === id);
    if (!tab || !tab.url || tab.url === "") {
      set({ invokeLoading: false });
      return true;
    }
    let rClone = structuredClone(tab);
    delete rClone.response;
    let t = cleanUpRequest(rClone);
    let rsp = await InvokeRequest(t);
    if (!rsp.success) {
      set({ invokeLoading: false });
      return false;
    }
    set((x) => {
      let tab = x.tabs.find((t) => t.id === id);
      if (tab) {
        tab.response = rsp.data;
      }
      x.invokeLoading = false;
    });
    return true;
  },
  openChoseFile: async (tabId, form_id) => {
    let rsp = await ChoseFile();
    if (!rsp.success || !rsp.data || !rsp.data.length) return;
    set((x) => {
      let t = x.tabs.find((t) => t.id === tabId);
      if (!t) return;
      let h = t.body.formData.find((h) => h.id === form_id);
      if (!h) return;
      h.value = "";
      h.files = rsp.data;
    });
  },
  tabInx: 0,
  setTabInx: (i) => set(() => ({ tabInx: i })),
  nextTab: () => {
    set((x) => {
      if (x.tabs.length === 0) return x;
      const nextIndex = (x.tabInx + 1) % x.tabs.length;
      return { tabInx: nextIndex };
    });
  },
  prevTab: () => {
    set((x) => {
      if (x.tabs.length === 0) return x;
      const prevIndex = (x.tabInx - 1 + x.tabs.length) % x.tabs.length;
      return { tabInx: prevIndex };
    });
  },
  setReqTabInx: (id, i) => {
    set((x) => {
      const t = x.tabs.find((t) => t.id === id);
      if (!t) return;
      t.reqTabInx = i;
    });
  },

  createTab: () => set((x) => ({ tabs: [...x.tabs, tabSchema({ name: "Untitled" })], tabInx: x.tabs.length })),

  openTab: async (t) => {
    let rsp = await GetRequest(t.id, t.coll_id);
    if (!rsp.success) return;
    set((x) => {
      const eInx = x.tabs.findIndex((tab) => tab.id === t.id);
      if (eInx !== -1) {
        x.tabInx = eInx;
      } else {
        return { tabs: [...x.tabs, rsp.data], tabInx: x.tabs.length };
      }
    });
  },

  updateTab: (id, key, value) =>
    set((x) => {
      let t = x.tabs.find((t) => t.id === id);
      if (!t) return;
      t[key] = value;
    }),

  updateReqBody: (id, key, value) =>
    set((x) => {
      let t = x.tabs.find((t) => t.id === id);
      if (!t) return;
      t.body[key] = value;
    }),

  deleteTab: (id) =>
    set((x) => {
      const tabIndex = x.tabs.findIndex((tab) => tab.id === id);
      const cf = x.tabInx;
      if (tabIndex === -1) return x;
      const newTabs = x.tabs.filter((tab) => tab.id !== id);
      let nf = cf;
      if (cf === tabIndex) {
        nf = tabIndex > 0 ? tabIndex - 1 : 0;
      } else if (cf > tabIndex) {
        nf = cf - 1;
      }
      nf = Math.min(nf, newTabs.length - 1);
      return {
        tabs: newTabs,
        tabInx: newTabs.length > 0 ? nf : -1,
      };
    }),

  updateHeaders: (id, pId, key, value) =>
    set((x) => {
      let t = x.tabs.find((t) => t.id === id);
      if (!t) return;
      let h = t.headers.find((h) => h.id === pId);
      if (!h) return;
      h[key] = value;
    }),

  deleteHeaders: (id, pId) =>
    set((x) => {
      let t = x.tabs.find((t) => t.id === id);
      if (!t) return;
      t.headers = t.headers.filter((h) => h.id !== pId);
    }),

  addHeaders: (id) =>
    set((x) => {
      let t = x.tabs.find((t) => t.id === id);
      if (!t) return;
      if (t.headers === null || !t.headers.length) {
        t.headers = [{ id: nanoid(), key: "", value: "", active: true }];
      } else {
        t.headers.push({ id: nanoid(), key: "", value: "", active: true });
      }
    }),

  updateParams: (id, pId, key, value) =>
    set((x) => {
      let t = x.tabs.find((t) => t.id === id);
      if (!t) return;
      let h = t.params.find((h) => h.id === pId);
      if (!h) return;
      h[key] = value;
    }),

  deleteParam: (id, pId) =>
    set((x) => {
      let t = x.tabs.find((t) => t.id === id);
      if (!t) return;
      t.params = t.params.filter((h) => h.id !== pId);
    }),

  addParam: (id) =>
    set((x) => {
      let t = x.tabs.find((t) => t.id === id);
      if (!t) return;
      if (t.params === null || !t.params.length) {
        t.params = [{ id: nanoid(), key: "", value: "", active: true }];
      } else {
        t.params.push({ id: nanoid(), key: "", value: "", active: true });
      }
    }),

  updateFormData: (id, pId, key, value) =>
    set((x) => {
      let t = x.tabs.find((t) => t.id === id);
      if (!t) return;
      let h = t.body.formData.find((h) => h.id === pId);
      if (!h) return;
      h[key] = value;
      if (key === "type") {
        if (value === "file") h.value = "";
        if (value === "text") h.files = [];
      }
    }),
  deleteFormData: (id, pId) =>
    set((x) => {
      let t = x.tabs.find((t) => t.id === id);
      if (!t) return;
      t.body.formData = t.body.formData.filter((h) => h.id !== pId);
    }),
  addFormData: (id) =>
    set((x) => {
      let t = x.tabs.find((t) => t.id === id);
      if (!t) return;
      t.body.formData.push({ id: nanoid(), key: "", value: "", files: [], type: "text", active: true });
    }),
});
