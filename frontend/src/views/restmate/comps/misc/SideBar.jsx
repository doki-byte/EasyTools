import { useState } from "react";
import { nanoid } from "nanoid";
import { LuFolder, LuPlus, LuCog, LuContainer } from "react-icons/lu";
import ModalLayout from "./ModalLayout";
import CustomButton from "./CustomButton";
import { useStore } from "../../store/store";
import Collection from "../collections/Collection";
import Tippy from "@tippyjs/react";
import EnvVar from "../envars/EnvVar";
import SettingModal from "../settings/SettingModal";
import { useHotkeys } from "react-hotkeys-hook";

const SideBar = () => {
  const [newColModal, setnewColModal] = useState(false);
  const [newEnvModal, setnewEnvModal] = useState(false);
  const [settingModal, setsettingModal] = useState(false);
  let cLoading = useStore((x) => x.cLoading);
  let envLoading = useStore((x) => x.envLoading);
  let envs = useStore((x) => x.envs);
  let cols = useStore((x) => x.collections);
  let sideBarType = useStore((x) => x.sideBarType);
  useHotkeys("ctrl+b", () => useStore.getState().setSideBar("col"), { enableOnFormTags: ["input", "select", "textarea"] });
  useHotkeys("ctrl+n", () => setnewColModal(true), { enableOnFormTags: ["input", "select", "textarea"] });
  let closeW = "50px";
  let openW = "250px";

  const createNewEnv = async (e) => {
    e.preventDefault();
    let name = e.target.env_name.value;
    if (!name) return;
    let success = await useStore.getState().addEnv(name);
    if (success) {
      setnewEnvModal(false);
    }
  };
  const createNewCollection = async (e) => {
    e.preventDefault();
    let c = {
      id: nanoid(),
      name: e.target.coll_name.value,
    };
    let success = await useStore.getState().addCols(c);
    if (success) {
      setnewColModal(false);
    }
  };
  return (
    <div className="w-full h-full flex">
      <div style={{ width: closeW }} className="h-full border-r border-lines" id="sidebar-main">
        <div className="h-full pb-2">
          <Tippy content="集合" delay="300">
            <div
              className={`${sideBarType === "col" ? "text-lit bg-sec" : "text-txtsec"} w-full flex justify-center items-center cursor-pointer hover:bg-sec hover:text-lit`}
              style={{ height: "48px" }}
              onClick={() => useStore.getState().setSideBar("col")}
            >
              <LuFolder size="22" />
            </div>
          </Tippy>
          <Tippy content="环境变量" delay="300">
            <div
              className={`${sideBarType === "env" ? "text-lit bg-sec" : "text-txtsec"} w-full flex justify-center items-center cursor-pointer hover:bg-sec hover:text-lit`}
              style={{ height: "48px" }}
              onClick={() => useStore.getState().setSideBar("env")}
            >
              <LuContainer size="22" />
            </div>
          </Tippy>
          <Tippy content="设置" delay="300">
            <div
              className="text-txtsec w-full flex justify-center items-center cursor-pointer hover:bg-sec hover:text-lit"
              style={{ height: "48px" }}
              onClick={() => setsettingModal(true)}
            >
              <LuCog size="22" />
            </div>
          </Tippy>
        </div>
      </div>
      <div style={{ width: sideBarType === "col" ? openW : 0 }} className="overflow-hidden h-full" id="colls-bar">
        <div className="border-r border-lines w-full h-full">
          <div className="text-txtprim flex justify-between items-center p-2 border-b border-lines" style={{ height: "48px" }} id="colls-topbar">
            <Tippy content="新建集合" delay="300">
              <div className="flex justify-start items-center gap-x-0.5 cursor-pointer hover:text-lit" onClick={() => setnewColModal(true)}>
                <LuPlus size="18" />
                <p className="text-sm">New</p>
              </div>
            </Tippy>
            <Tippy content="导入集合" delay="300">
              <div
                className="text-xs text-txtprim bg-sec py-1 px-2 rounded-sm cursor-pointer hover:bg-gray-600 hover:text-lit"
                onClick={() => useStore.getState().importCollection()}
              >
                <p>Import</p>
              </div>
            </Tippy>
          </div>
          <div id="colls-map" className="py-2 overflow-y-auto" style={{ height: "calc(100% - 48px)" }}>
            {cols?.length ? cols.map((c) => <Collection key={c.id} col={c} />) : null}
          </div>
        </div>
      </div>
      <div style={{ width: sideBarType === "env" ? openW : 0 }} className="overflow-hidden h-full" id="env-bar">
        <div className="border-r border-lines w-full h-full">
          <div className="text-txtprim flex justify-between items-center p-2 border-b border-lines" style={{ height: "48px" }} id="colls-topbar">
            <Tippy content="新的环境变量" delay="300">
              <div className="flex justify-start items-center gap-x-0.5 cursor-pointer hover:text-lit" onClick={() => setnewEnvModal(true)}>
                <LuPlus size="18" />
                <p className="text-sm">New</p>
              </div>
            </Tippy>
          </div>
          <div id="env-map" className="py-2 overflow-y-auto" style={{ height: "calc(100% - 48px)" }}>
            {envs?.length ? envs.map((e) => <EnvVar key={e.id} env={e} />) : null}
          </div>
        </div>
      </div>
      {settingModal && <SettingModal settingModal={settingModal} setsettingModal={setsettingModal} />}
      <ModalLayout open={newColModal} onClose={() => setnewColModal(false)} title="创建新的集合">
        <form onSubmit={createNewCollection}>
          <div className="p-6">
            <p className="text-txtprim text-sm mb-2">集合名称</p>
            <input
              name="coll_name"
              className="border border-txtsec text-lit w-full outline-none p-1 px-3 text-lg focus:border-txtprim rounded-sm"
              required
              maxLength={100}
              autoFocus
              autoComplete="off"
            />
            <div className="w-full flex justify-end items-center mt-6 gap-x-4">
              <CustomButton name="创建" type="submit" loading={cLoading} clx="px-4 py-1" />
              <CustomButton name="关闭" bg="bg-txtsec" clx="px-4 py-1" onClick={() => setnewColModal(false)} />
            </div>
          </div>
        </form>
      </ModalLayout>
      <ModalLayout open={newEnvModal} onClose={() => setnewEnvModal(false)} title="创建新的环境变量">
        <form onSubmit={createNewEnv}>
          <div className="p-6">
            <p className="text-txtprim text-sm mb-2">环境变量名称</p>
            <input
              name="env_name"
              className="border border-txtsec text-lit w-full outline-none p-1 px-3 text-lg focus:border-txtprim rounded-sm"
              required
              maxLength={100}
              autoFocus
              autoComplete="off"
            />
            <div className="w-full flex justify-end items-center mt-6 gap-x-4">
              <CustomButton name="创建" type="submit" loading={envLoading} clx="px-4 py-1" />
              <CustomButton name="关闭" bg="bg-txtsec" clx="px-4 py-1" onClick={() => setnewEnvModal(false)} />
            </div>
          </div>
        </form>
      </ModalLayout>
    </div>
  );
};

export default SideBar;
