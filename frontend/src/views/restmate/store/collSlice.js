import {
  DeleteRequest,
  DuplicateRequest,
  ImportCollection,
  MoveRequest,
  DeleteCollection,
  RenameRequest,
  AddCollection,
  GetCollections,
  RenameCollection,
  UpsertRequest
} from "../../../../wailsjs/go/restmate/RestMate";
import { nanoid } from "nanoid";
import { tabSchema } from "./tabSlice";
import { cleanUpRequest } from "../utils/utils";
import { toast } from "react-toastify";

export const createColSlice = (set, get) => ({
  collections: [],
  tabs: [tabSchema()],
  tabInx: 0,
  cLoading: false,
  saveLoad: false,

  getCollections: async () => {
    try {
      const rsp = await GetCollections();
      if (!rsp.success) throw new Error(rsp.msg);
      set({ collections: rsp.data });
      return rsp.data;
    } catch (err) {
      console.warn('[collSlice] getCollections failed', err);
      set({ collections: [] });
      return [];
    }
  },

  importCollection: async () => {
    set({ cLoading: true });
    const rsp = await ImportCollection();
    set({ cLoading: false });
    if (!rsp.success) {
      toast.error(rsp.msg);
      return;
    }
    set({ collections: rsp.data });
    toast.success(rsp.msg);
  },

  addCols: async (c) => {
    set({ cLoading: true });
    const rsp = await AddCollection(c.id, c.name);
    set({ cLoading: false });
    if (!rsp.success) return false;
    set({ collections: rsp.data });
    return true;
  },

  addNewReqtoCol: async (coll_id) => {
    set({ saveLoad: true });
    let newTab = tabSchema({ coll_id });
    // 保证 response 不为空
    newTab.response = { body: "", headers: [] };
    const t = cleanUpRequest(newTab);
    const rsp = await UpsertRequest(t);
    set({ saveLoad: false });
    if (!rsp.success) return false;
    set((x) => ({
      collections: rsp.data,
      tabs: [...x.tabs, t],
      tabInx: x.tabs.length
    }));
    return true;
  },

  onDuplicateReq: async (coll_id, id) => {
    set({ cLoading: true });
    const rsp = await DuplicateRequest(coll_id, id);
    set({ cLoading: false });
    if (!rsp.success) return false;
    set({ collections: rsp.data });
    return true;
  },

  updateReq: async (id) => {
    set({ saveLoad: true });
    let tab = get().tabs.find((t) => t.id === id);
    if (!tab) return false;
    let rClone = structuredClone(tab);
    rClone.response = rClone.response || { body: "", headers: [] };
    const t = cleanUpRequest(rClone);
    const rsp = await UpsertRequest(t);
    set({ saveLoad: false });
    if (!rsp.success) return false;
    set({ collections: rsp.data });
    return true;
  },

  saveReq: async (id, name, coll_id) => {
    set({ saveLoad: true });
    let tab = get().tabs.find((t) => t.id === id);
    if (!tab) return false;
    let rClone = structuredClone(tab);
    rClone.response = rClone.response || { body: "", headers: [] };
    const t = cleanUpRequest(rClone);
    t.name = name;
    t.coll_id = coll_id;
    const rsp = await UpsertRequest(t);
    if (!rsp.success) {
      set({ saveLoad: false });
      return false;
    }
    set((x) => {
      x.collections = rsp.data;
      const tabIdx = x.tabs.findIndex(t => t.id === id);
      if (tabIdx > -1) {
        x.tabs[tabIdx].name = name;
        x.tabs[tabIdx].coll_id = coll_id;
      }
      x.saveLoad = false;
    });
    return true;
  },

  renameCollection: async (id, name) => {
    set({ cLoading: true });
    const rsp = await RenameCollection(id, name);
    set({ cLoading: false });
    if (!rsp.success) return false;
    set({ collections: rsp.data });
    return true;
  },

  renameReq: async (coll_id, id, name) => {
    set({ cLoading: true });
    const rsp = await RenameRequest(coll_id, id, name);
    set({ cLoading: false });
    if (!rsp.success) return false;
    set((x) => {
      const tab = x.tabs.find(t => t.id === id);
      if (tab) tab.name = name;
      x.collections = rsp.data;
    });
    return true;
  },

  deleteReq: async (coll_id, id) => {
    set({ cLoading: true });
    const rsp = await DeleteRequest(coll_id, id);
    set({ cLoading: false });
    if (!rsp.success) return false;
    set((x) => {
      x.collections = rsp.data;
      x.tabs = x.tabs.filter(t => t.id !== id);
    });
    return true;
  },

  deleteCol: async (coll_id) => {
    set({ cLoading: true });
    const rsp = await DeleteCollection(coll_id);
    set({ cLoading: false });
    if (!rsp.success) return false;
    set((x) => ({
      collections: rsp.data,
      tabs: x.tabs.filter(t => t.coll_id !== coll_id)
    }));
    return true;
  },

  moveReq: async (id, coll_id, new_coll_id) => {
    set({ cLoading: true });
    const new_id = nanoid();
    const rsp = await MoveRequest(id, new_id, coll_id, new_coll_id);
    set({ cLoading: false });
    if (!rsp.success) return false;
    set((x) => {
      x.collections = rsp.data;
      const tab = x.tabs.find(t => t.id === id);
      if (tab) {
        tab.id = new_id;
        tab.coll_id = new_coll_id;
      }
    });
    return true;
  },
});
