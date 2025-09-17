import { Tab, Tabs, TabList, TabPanel } from "react-tabs";
import "react-tabs/style/react-tabs.css";
import { useEffect, useRef } from "react";
import { LuChevronLeft, LuChevronRight, LuPlus, LuX } from "react-icons/lu";
import TabPanelRoot from "./TabPanelRoot";
import { useStore } from "../store/store";
import { getReqType } from "../utils/helper";
import EnvSelector from "./envars/EnvSelector";
import { useHotkeys } from "react-hotkeys-hook";

const TabsRoot = () => {
  const { tabs, deleteTab, createTab, tabInx, nextTab, prevTab } = useStore();
  useHotkeys("ctrl+t", () => createTab(), { enableOnFormTags: ["input", "select", "textarea"] });
  useHotkeys("ctrl+right", () => nextTab(), { enableOnFormTags: ["input", "select", "textarea"] });
  useHotkeys("ctrl+left", () => prevTab(), { enableOnFormTags: ["input", "select", "textarea"] });
  const tabsWrapper = useRef(null);
  const scrolLeftBtn = useRef(null);
  const scrolRightBtn = useRef(null);

  useEffect(() => {
    const tw = tabsWrapper.current;
    const right = scrolRightBtn.current;
    const left = scrolLeftBtn.current;
    if (tw.scrollWidth > tw.clientWidth) {
      left.style.display = "block";
      right.style.display = "block";
    } else {
      left.style.display = "none";
      right.style.display = "none";
    }
  }, [tabs]);

  const handleWheel = (e) => {
    const container = tabsWrapper.current;
    const scrollAmount = 400;
    if (e.deltaX !== 0) {
      container.scrollBy({
        left: e.deltaX > 0 ? scrollAmount : -scrollAmount,
        behavior: "smooth",
      });
    }
    if (e.deltaY !== 0) {
      container.scrollBy({
        left: e.deltaY > 0 ? scrollAmount : -scrollAmount,
        behavior: "smooth",
      });
    }
  };

  const scrollTabs = (amount) => {
    const tabsWp = tabsWrapper.current;
    tabsWp.scrollBy({
      left: amount,
      behavior: "smooth",
    });
  };
  return (
    <Tabs style={{ height: "100%" }} selectedIndex={tabInx} onSelect={(i) => useStore.getState().setTabInx(i)} disableLeftRightKeys={true}>
      <div className="grid h-full" style={{ gridTemplateRows: "48px minmax(0,100%)", gridTemplateColumns: "100%" }}>
        <div className="w-full h-full flex justify-between items-center bg-sec">
          <div className="flex items-center h-full min-w-0">
            <div
              onClick={() => scrollTabs(-400)}
              className="cursor-pointer bg-sec text-lit p-2 hover:text-accent border-r border-lines hidden"
              ref={scrolLeftBtn}
            >
              <LuChevronLeft size="20" />
            </div>
            <div className="w-fit h-full overflow-x-hidden" ref={tabsWrapper} onWheel={(e) => handleWheel(e)}>
              <TabList className="flex items-center h-full">
                {tabs.map((t) => (
                  <Tab
                    key={t.id}
                    selectedClassName="!text-lit bg-brand !border-accent"
                    className="flex items-center justify-between gap-x-2 outline-none h-full px-2 text-txtsec border-t-2 border-sec cursor-pointer max-w-60 w-60 min-w-40 group"
                  >
                    <div className="text-xs">{getReqType(t.method)}</div>
                    <div className="grow overflow-hidden">
                      <p className="truncate whitespace-nowrap overflow-ellipsis">{t.name}</p>
                    </div>
                    <div
                      onClick={(e) => {
                        e.stopPropagation();
                        deleteTab(t.id);
                      }}
                      className="hidden p-1 rounded-md text-txtsec items-center hover:text-lit hover:bg-gray-700 group-hover:flex"
                    >
                      <LuX />
                    </div>
                  </Tab>
                ))}
              </TabList>
            </div>
            <div className="flex items-center">
              <div
                onClick={() => scrollTabs(400)}
                className="cursor-pointer bg-sec text-lit p-2 hover:text-accent border-x border-lines h-full hidden"
                ref={scrolRightBtn}
              >
                <LuChevronRight size="20" />
              </div>
              <div onClick={createTab} className="cursor-pointer bg-sec text-lit p-2 hover:text-accent">
                <LuPlus size="20" />
              </div>
            </div>
          </div>
          <EnvSelector />
        </div>
        <div className="h-full">
          {tabs.map((t) => (
            <TabPanel key={t.id} style={{ height: "100%" }}>
              <TabPanelRoot tab={t} />
            </TabPanel>
          ))}
        </div>
      </div>
    </Tabs>
  );
};

export default TabsRoot;
