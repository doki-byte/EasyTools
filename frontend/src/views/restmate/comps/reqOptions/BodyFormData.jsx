import React, { useEffect } from "react";
import { LuCircle, LuCircleCheckBig, LuTrash2 } from "react-icons/lu";
import { useStore } from "../../store/store";
import { Menu, MenuItem } from "@szhsin/react-menu";
import { LuChevronDown } from "react-icons/lu";
import DraftEditor from "../misc/DraftEditor";

const BodyFormData = ({ tabId, formData, envVars }) => {
  const updateFormData = useStore((x) => x.updateFormData);
  const deleteFormData = useStore((x) => x.deleteFormData);
  const addFormData = useStore((x) => x.addFormData);

  const onChoseFile = async (form_id) => {
    await useStore.getState().openChoseFile(tabId, form_id);
  };
  useEffect(() => {
    if (formData && formData.length) {
      let last = formData[formData.length - 1];
      if (last && last.key !== "") {
        addFormData(tabId);
      }
    } else {
      addFormData(tabId);
    }
  }, [formData]);
  return (
    <div className="h-full grid" style={{ gridTemplateRows: "minmax(0,100%)" }}>
      {formData && formData.length ? (
        <div className="pt-2 overflow-y-auto overflow-x-hidden">
          {formData.map((p, i) => (
            <div key={p.id} className="flex items-center border border-b-0 border-lines last:border-b h-8">
              <div className="border-r border-lines h-full basis-1/2">
                <input
                  value={p.key}
                  className="outline-none text-txtprim px-2 w-full h-full focus:text-lit focus:bg-sec"
                  placeholder="key"
                  maxLength="99"
                  onChange={(e) => updateFormData(tabId, p.id, "key", e.target.value)}
                />
              </div>
              <div className="border-r border-lines grow h-full">
                <Menu
                  menuButton={
                    <button className="px-4 w-20 h-full cursor-pointer flex justify-start items-center gap-x-1 text-txtsec text-sm capitalize">
                      {p.type}
                      <LuChevronDown size="16" />
                    </button>
                  }
                  menuClassName="!bg-sec"
                  unmountOnClose={true}
                  align="start"
                  direction="bottom"
                  gap={6}
                >
                  <MenuItem className="text-txtprim" onClick={() => updateFormData(tabId, p.id, "type", "text")}>
                    Text
                  </MenuItem>
                  <MenuItem className="text-txtprim" onClick={() => updateFormData(tabId, p.id, "type", "file")}>
                    File
                  </MenuItem>
                </Menu>
              </div>
              <div className="h-full basis-1/2">
                {/*
                  <input
                    value={p.value}
                    className="outline-none text-txtprim px-2 w-full h-full focus:text-lit focus:bg-sec"
                    placeholder="value"
                    maxLength="999"
                    onChange={(e) => updateFormData(tabId, p.id, "value", e.target.value)}
                  />
                  */}
                {p.type === "text" ? (
                  <DraftEditor value={p.value} setValue={(e) => updateFormData(tabId, p.id, "value", e)} envVars={envVars} />
                ) : (
                  <div className="w-full h-full px-2 flex items-center cursor-pointer" onClick={() => onChoseFile(p.id)}>
                    {p.files && p.files?.length ? (
                      <p className="text-sm text-txtprim bg-sec px-4 py-1 rounded-full">
                        {p.files?.length} {p.files?.length > 1 ? "files" : "file"} selected
                      </p>
                    ) : (
                      <p className="text-sm text-txtsec">选择文件</p>
                    )}
                  </div>
                )}
              </div>
              <div
                className="h-full flex items-center px-2 hover:bg-sec cursor-pointer border-x border-lines"
                onClick={() => updateFormData(tabId, p.id, "active", !p.active)}
              >
                {p.active ? <LuCircleCheckBig className="text-green-500" /> : <LuCircle className="text-txtsec" />}
              </div>
              {formData.length === i + 1 ? (
                <div className="h-full flex items-center px-2 hover:bg-sec">
                  <LuTrash2 className="text-txtsec" />
                </div>
              ) : (
                <div className="h-full flex items-center px-2 hover:bg-sec cursor-pointer" onClick={() => deleteFormData(tabId, p.id)}>
                  <LuTrash2 className="text-red-500" />
                </div>
              )}
            </div>
          ))}
        </div>
      ) : null}
    </div>
  );
};

export default React.memo(BodyFormData);
