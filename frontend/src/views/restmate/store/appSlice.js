export const createAppSlice = (set) => ({
  appLoading: false,
  sideBarType: null,
  cookieModal: false,
  setCookieModal: (s) => set({ cookieModal: s }),
  setSideBar: (s) => {
    set((x) => {
      if (x.sideBarType === s) {
        x.sideBarType = null;
      } else {
        x.sideBarType = s;
      }
    });
  },
});
