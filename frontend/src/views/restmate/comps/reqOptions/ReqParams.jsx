import React, { useEffect } from "react";
import { LuCircle, LuCircleCheckBig, LuTrash2 } from "react-icons/lu";
import { useStore } from "../../store/store";
import DraftEditor from "../misc/DraftEditor";

const ReqParams = ({ tabId, params, envVars }) => {
  const updateParam = useStore((x) => x.updateParams);
  const deleteParam = useStore((x) => x.deleteParam);
  const addParam = useStore((x) => x.addParam);
  useEffect(() => {
    if (params && params.length) {
      let last = params[params.length - 1];
      if (last && last.key !== "") {
        addParam(tabId);
      }
    } else {
      addParam(tabId);
    }
  }, [params]);
  return (
    <div className="pt-2 h-full grid" style={{ gridTemplateRows: "min-content minmax(0,100%)" }}>
      <div className="flex items-center justify-between">
        <p className="text-txtsec text-sm font-bold">Query Params</p>
      </div>
      {params && params.length ? (
        <div className="pt-2 overflow-y-auto overflow-x-hidden">
          {params.map((p, i) => (
            <div key={p.id} className="flex items-center border border-b-0 border-lines last:border-b h-8">
              <div className="border-r border-lines h-full basis-1/2">
                <input
                  value={p.key}
                  className="outline-none text-txtprim text-sm px-2 w-full h-full focus:text-lit focus:bg-sec"
                  placeholder="key"
                  maxLength="99"
                  onChange={(e) => updateParam(tabId, p.id, "key", e.target.value)}
                />
              </div>
              <div className="h-full basis-1/2">
                <DraftEditor value={p.value} setValue={(e) => updateParam(tabId, p.id, "value", e)} envVars={envVars} />
              </div>
              <div
                className="h-full flex items-center px-2 hover:bg-sec cursor-pointer border-x border-lines"
                onClick={() => updateParam(tabId, p.id, "active", !p.active)}
              >
                {p.active ? <LuCircleCheckBig className="text-green-500" /> : <LuCircle className="text-txtsec" />}
              </div>
              {params.length === i + 1 ? (
                <div className="h-full flex items-center px-2 hover:bg-sec">
                  <LuTrash2 className="text-txtsec" />
                </div>
              ) : (
                <div className="h-full flex items-center px-2 hover:bg-sec cursor-pointer" onClick={() => deleteParam(tabId, p.id)}>
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

export default React.memo(ReqParams);
