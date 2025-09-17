import React from "react";
import { useStore } from "../../store/store";
import { Menu, MenuItem } from "@szhsin/react-menu";
import { LuChevronDown } from "react-icons/lu";

const ReqBodyOption = ({ tabId, bodyType }) => {
  const updateReqBody = useStore((x) => x.updateReqBody);
  return (
    <div className="">
      <Menu
        menuButton={
          <button className="shrink-0 h-full cursor-pointer flex justify-center items-center gap-x-1 text-txtsec text-sm font-bold uppercase">
            {bodyType}
            <LuChevronDown size="16" />
          </button>
        }
        menuClassName="!bg-sec"
        unmountOnClose={false}
        align="start"
        direction="bottom"
        gap={6}
      >
        <MenuItem className="text-txtprim text-sm" onClick={() => updateReqBody(tabId, "bodyType", "json")}>
          JSON
        </MenuItem>
        <MenuItem className="text-txtprim text-sm" onClick={() => updateReqBody(tabId, "bodyType", "formdata")}>
          FormData
        </MenuItem>
        <MenuItem className="text-txtprim text-sm" onClick={() => updateReqBody(tabId, "bodyType", "none")}>
          None
        </MenuItem>
      </Menu>
    </div>
  );
};

export default React.memo(ReqBodyOption);
