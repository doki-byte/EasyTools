import { Menu, MenuItem } from "@szhsin/react-menu";
import { useStore } from "../../store/store";
import React from "react";
import { LuCaptionsOff } from "react-icons/lu";
import { toast } from "react-toastify";
const EnvSelector = () => {
  const envs = useStore((x) => x.envs);
  const selectedEnv = envs && envs.length ? envs.find((e) => e.selected === true) : null;

  const onSelectEnv = async (id) => {
    let rsp = await useStore.getState().setSelectedEnv(id);
    if (!rsp) {
      toast.error("错误！无法选择环境。");
    }
  };

  return (
    <div className="h-full w-40 shrink-0 border-l border-lines bg-sec text-txtprim hover:bg-brand hover:text-lit">
      <Menu
        menuButton={
          <div className="cursor-pointer w-full h-full flex items-center justify-center px-4 gap-x-2">
            {selectedEnv ? (
              <p className="text-xs truncate overflow-ellipsis">{selectedEnv?.name}</p>
            ) : (
              <>
                <LuCaptionsOff />
                <p className="text-xs truncate overflow-ellipsis">No Environment</p>
              </>
            )}
          </div>
        }
        menuClassName="!bg-sec !border !border-lines"
        unmountOnClose={false}
        align="start"
        direction="bottom"
        gap={0}
      >
        <MenuItem className="text-txtsec text-xs gap-x-2" onClick={() => onSelectEnv("none")}>
          <LuCaptionsOff />
          No Environment
        </MenuItem>
        {envs && envs?.length ? (
          envs.map((x) => (
            <React.Fragment key={x.id}>
              <MenuItem onClick={() => onSelectEnv(x.id)}>
                <p className="text-txtprim text-xs truncate overflow-ellipsis" style={{ maxWidth: "300px" }}>
                  {x.name}
                </p>
              </MenuItem>
            </React.Fragment>
          ))
        ) : (
          <div className="px-6 py-2 text-xs text-txtprim">
            <p>未找到环境。</p>
            <p>请创建一个新环境。</p>
          </div>
        )}
      </Menu>
    </div>
  );
};

export default EnvSelector;
