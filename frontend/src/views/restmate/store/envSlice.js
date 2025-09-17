import { AddEnv, AddVar, DeleteEnv, DeleteVar, DuplicateEnv, GetEnvs, RenameEnv, SelectEnv } from "../../../../wailsjs/go/restmate/RestMate";

export const createEnvSlice = (set) => ({
  envs: [],
  envLoading: false,

  setSelectedEnv: async (id) => {
    set({ envLoading: true });
    const rsp = await SelectEnv(id);
    set({ envLoading: false });
    if (!rsp.success) return false;
    set({ envs: rsp.data });
    return true;
  },

  renameEnv: async (id, name) => {
    set({ envLoading: true });
    const rsp = await RenameEnv(id, name);
    set({ envLoading: false });
    if (!rsp.success) return false;
    set({ envs: rsp.data });
    return true;
  },

  addEnv: async (name) => {
    set({ envLoading: true });
    const rsp = await AddEnv(name);
    set({ envLoading: false });
    if (!rsp.success) return false;
    set({ envs: rsp.data });
    return true;
  },

  duplicateEnv: async (id) => {
    set({ envLoading: true });
    const rsp = await DuplicateEnv(id);
    set({ envLoading: false });
    if (!rsp.success) return false;
    set({ envs: rsp.data });
    return true;
  },

  deleteEnv: async (id) => {
    set({ envLoading: true });
    const rsp = await DeleteEnv(id);
    set({ envLoading: false });
    if (!rsp.success) return false;
    set({ envs: rsp.data });
    return true;
  },

  getEnvs: async () => {
    set({ envLoading: true });
    const rsp = await GetEnvs();
    set({ envLoading: false });
    if (!rsp.success) {
      set({ envs: [] });
      return { envs: [] };
    }
    set({ envs: rsp.data ?? [] });
    return rsp.data ?? [];
  },

  addNewVar: async (id, k, v) => {
    set({ envLoading: true });
    const rsp = await AddVar(id, k, v);
    set({ envLoading: false });
    if (!rsp.success) return false;
    set({ envs: rsp.data });
    return true;
  },

  deleteVar: async (id, name) => {
    set({ envLoading: true });
    const rsp = await DeleteVar(id, name);
    set({ envLoading: false });
    if (!rsp.success) return false;
    set({ envs: rsp.data });
    return true;
  },
});
