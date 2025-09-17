import { LuCopy, LuEllipsis, LuExternalLink, LuMove, LuPencil, LuTrash } from "react-icons/lu";
import { useStore } from "../../store/store";
import { getReqType } from "../../utils/helper";
import { Menu, MenuItem } from "@szhsin/react-menu";
import { useState } from "react";
import RenameReq from "./RenameReq";
import MoveReq from "./MoveReq";
import { toast } from "react-toastify";

const RequestList = ({ req }) => {
  const [renameModal, setRenameModal] = useState(false);
  const [moveReqModal, setmoveReqModal] = useState(false);
  const onDeleteReq = async () => {
    let rsp = await useStore.getState().deleteReq(req.coll_id, req.id);
    if (rsp) {
      toast.success("请求删除成功！");
    } else {
      toast.error("错误！无法删除请求。");
    }
  };
  return (
    <div key={req.id} className="text-txtprim hover:bg-sec hover:text-lit pl-7 py-1 cursor-pointer group flex items-center">
      <div className="grow overflow-hidden flex items-center" onClick={() => useStore.getState().openTab(req)}>
        <div className="mr-2 text-xs">{getReqType(req.method)}</div>
        <p className="truncate whitespace-nowrap overflow-ellipsis text-sm" style={{ width: "90%" }}>
          {req.name}
        </p>
      </div>
      <Menu
        menuButton={({ open }) => (
          <div className={`${open ? "block" : "hidden"} group-hover:block pr-2`}>
            <div className="cursor-pointer hover:text-lit">
              <LuEllipsis size="20" />
            </div>
          </div>
        )}
        menuClassName="!bg-sec"
        unmountOnClose={false}
        align="start"
        direction="bottom"
        gap={0}
      >
        <MenuItem className="text-txtprim text-sm gap-x-2" onClick={() => useStore.getState().openTab(req)}>
          <LuExternalLink />
          在新标签打开
        </MenuItem>
        <MenuItem className="text-txtprim text-sm gap-x-2" onClick={() => setRenameModal(true)}>
          <LuPencil />
          重命名
        </MenuItem>
        <MenuItem className="text-txtprim text-sm gap-x-2" onClick={() => useStore.getState().onDuplicateReq(req.coll_id, req.id)}>
          <LuCopy />
          复制
        </MenuItem>
        <MenuItem className="text-txtprim text-sm gap-x-2" onClick={() => setmoveReqModal(true)}>
          <LuMove />
          移动
        </MenuItem>
        <MenuItem className="text-red-400 text-sm gap-x-2" onClick={() => onDeleteReq()}>
          <LuTrash />
          删除
        </MenuItem>
      </Menu>
      {renameModal && <RenameReq renameModal={renameModal} setRenameModal={setRenameModal} req={req} />}
      {moveReqModal && <MoveReq moveModal={moveReqModal} setmoveModal={setmoveReqModal} req_id={req.id} coll_id={req.coll_id} />}
    </div>
  );
};

export default RequestList;
