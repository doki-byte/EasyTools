import React, { useState } from "react";
import { useStore } from "../store/store";
import { Menu, MenuItem } from "@szhsin/react-menu";
import { getReqType } from "../utils/helper";
import { LuChevronDown, LuInfo, LuRadio, LuSave } from "react-icons/lu";
import CustomButton from "./misc/CustomButton";
import ModalLayout from "./misc/ModalLayout";
import { toast } from "react-toastify";
import DraftEditor from "./misc/DraftEditor";
import { EventsEmit } from "../../../../wailsjs/runtime";
import { useHotkeys } from "react-hotkeys-hook";

const ReqHead = ({ tabId, method, url, name, coll_id, envVars }) => {
  const [saveModal, setSaveModal] = useState(false);
  const [selcol, setSelcol] = useState(coll_id);
  let saveLoad = useStore((x) => x.saveLoad);
  let invokeLoading = useStore((x) => x.invokeLoading);
  let cols = useStore((x) => x.collections);
  useHotkeys("ctrl+s", () => updateReqModal(), { enableOnFormTags: ["input", "select", "textarea"] });
  useHotkeys("ctrl+enter", () => onInvokeReq(), { enableOnFormTags: ["input", "select", "textarea"] });
  useHotkeys("ctrl+w", () => useStore.getState().deleteTab(tabId), { enableOnFormTags: ["input", "select", "textarea"] });
  const updateReqModal = async () => {
    if (!coll_id || coll_id === "") {
      setSelcol(coll_id);
      setSaveModal(true);
    } else {
      let rsp = await useStore.getState().updateReq(tabId);
      if (rsp) {
        toast.success("请求保存成功！");
      }
    }
  };
  const getColsName = () => {
    if (coll_id) {
      let c = cols && cols.find((x) => x.id === coll_id);
      if (!c) return name;
      return c.name + " / " + name;
    }
    return name;
  };
  const onUpdateReq = async (e) => {
    e.preventDefault();
    let n = e.target.req_name.value;
    if (!n || n === "") {
      toast.warn("名称不能为空。");
      return;
    }
    if (!selcol || selcol === "") {
      toast.warn("请选择集合。");
      return;
    }
    let rsp = await useStore.getState().saveReq(tabId, n, selcol);
    if (rsp) {
      setSaveModal(false);
      toast.success("请求保存成功！");
    } else {
      toast.error("错误！无法保存请求。");
    }
  };
  const onInvokeReq = async () => {
    let rsp = await useStore.getState().invokeReq(tabId);
    if (!rsp) {
      toast.error("错误！请求失败。");
    }
  };

  return (
    <div className="h-full px-6">
      <div className="flex justify-between items-center gap-x-x">
        <div className="flex items-center text-accent gap-x-2 grow">
          <LuRadio />
          <p className="text-sm text-txtprim max-w-2/3 truncate text-ellipsis">{getColsName()}</p>
        </div>
      </div>
      <div className="mt-4">
        <div className="flex items-center gap-x-3 h-10">
          <div className="grow border border-txtsec flex justify-start items-center rounded-sm h-full">
            <Menu
              menuButton={
                <button className="w-28 shrink-0 h-full cursor-pointer flex justify-center items-center gap-x-4 text-txtsec border-r border-txtsec">
                  {getReqType(method)}
                  <LuChevronDown size="22" />
                </button>
              }
              menuClassName="!bg-sec"
              unmountOnClose={false}
              align="start"
              direction="bottom"
              gap={6}
            >
              <MenuItem className="text-green-400 font-bold" onClick={() => useStore.getState().updateTab(tabId, "method", "get")}>
                GET
              </MenuItem>
              <MenuItem className="text-yellow-400 font-bold" onClick={() => useStore.getState().updateTab(tabId, "method", "post")}>
                POST
              </MenuItem>
              <MenuItem className="text-blue-400 font-bold" onClick={() => useStore.getState().updateTab(tabId, "method", "put")}>
                PUT
              </MenuItem>
              <MenuItem className="text-red-400 font-bold" onClick={() => useStore.getState().updateTab(tabId, "method", "delete")}>
                DELETE
              </MenuItem>
            </Menu>
            <DraftEditor envVars={envVars} value={url} setValue={(e) => useStore.getState().updateTab(tabId, "url", e)} fontsm={false} invoke={onInvokeReq} />
          </div>
          {invokeLoading ? (
            <div className="h-full">
              <CustomButton clx="h-full px-6" bg="bg-txtsec" name="取消" onClick={() => EventsEmit("cancelRequest")} />
            </div>
          ) : (
            <div className="h-full">
              <CustomButton clx="h-full px-8" name="发送" onClick={onInvokeReq} />
            </div>
          )}
          <div className="h-full">
            <CustomButton clx="h-full px-6" bg="bg-txtsec" loading={saveLoad} name={<LuSave size="20" />} onClick={updateReqModal} />
          </div>
        </div>
      </div>
      <ModalLayout open={saveModal} onClose={() => setSaveModal(false)} title="Save Request">
        <form onSubmit={onUpdateReq}>
          <div className="p-6">
            <p className="text-txtprim text-sm mb-2">Request Name</p>
            <input
              name="req_name"
              className="border border-lines text-lit w-full outline-none p-1 px-3 text-lg focus:border-txtprim rounded-sm"
              defaultValue={name}
              required
              maxLength={100}
              autoFocus
              autoComplete="off"
            />
            <div className="mt-4">
              <p className="text-txtprim text-sm">Select Collection</p>
            </div>
            {cols && cols.length ? (
              <div className="bg-sec mt-2 border border-lines overflow-y-auto" style={{ maxHeight: "300px" }}>
                {cols.map((x) => (
                  <div
                    className={`${x.id === selcol ? "bg-txtprim text-brand" : "bg-sec text-txtprim"} rounded-sm p-2 cursor-pointer`}
                    key={x.id}
                    onClick={() => setSelcol(x.id)}
                  >
                    <p className="text-sm truncate whitespace-nowrap overflow-ellipsis" style={{ maxWidth: "90%" }}>
                      {x.name}
                    </p>
                  </div>
                ))}
              </div>
            ) : (
              <div className="bg-sec mt-2 p-4 border border-lines rounded-sm">
                <div className="flex justify-center mb-1 text-orange-400">
                  <LuInfo size="22" />
                </div>
                <p className="text-txtprim text-sm text-center">No collections found.</p>
                <p className="text-txtprim text-sm text-center">Please create a new collection first.</p>
              </div>
            )}
            <div className="w-full flex justify-end items-center mt-6 gap-x-4">
              <CustomButton name="保存" type="submit" loading={saveLoad} clx="px-4 py-1" />
              <CustomButton name="关闭" bg="bg-txtsec" clx="px-4 py-1" onClick={() => setSaveModal(false)} />
            </div>
          </div>
        </form>
      </ModalLayout>
    </div>
  );
};

export default React.memo(ReqHead);
