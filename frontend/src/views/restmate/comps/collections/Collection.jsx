import { useCollapse } from "react-collapsed";
import { LuChevronDown, LuChevronRight, LuDownload, LuEllipsis, LuPencil, LuPlus, LuTrash } from "react-icons/lu";
import { Menu, MenuItem } from "@szhsin/react-menu";
import RequestList from "./RequestList";
import { useState } from "react";
import RenameCol from "./RenameCol";
import { ExportCollection } from "../../../../../wailsjs/go/restmate/RestMate";
import { toast } from "react-toastify";
import { useStore } from "../../store/store";
import { memo } from "react";

const Collection = ({ col }) => {
  const [renameCol, setRenameCol] = useState(false);
  const { getCollapseProps, getToggleProps, isExpanded } = useCollapse();

  const exportCollection = async () => {
    let rsp = await ExportCollection(col.id);
    if (rsp.success) {
      toast.success("集合导出成功！");
    } else {
      toast.error("错误！无法导出集合。");
    }
  };
  const onDeleteCol = async () => {
    let rsp = await useStore.getState().deleteCol(col.id);
    if (rsp) {
      toast.success("已成功删除集合！");
    } else {
      toast.error("错误！无法删除集合。");
    }
  };
  return (
    <div className="text-txtprim">
      <div className={`${isExpanded ? "bg-sec text-lit" : ""} flex items-center py-1 hover:bg-sec hover:text-lit group`}>
        <div className="pl-2 pr-1 cursor-pointer" {...getToggleProps()}>
          {isExpanded ? <LuChevronDown size="18" /> : <LuChevronRight size="18" />}
        </div>
        <div className="grow overflow-hidden cursor-pointer" {...getToggleProps()}>
          <p className="truncate whitespace-nowrap overflow-ellipsis text-sm" style={{ width: "90%" }}>
            {col.name}
          </p>
        </div>
        <Menu
          menuButton={({ open }) => (
            <div className={`${open ? "block" : "hidden"} group-hover:block pr-2`}>
              <div className="cursor-pointer text-txtprim hover:text-lit">
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
          <MenuItem className="text-txtprim text-sm gap-x-2" onClick={() => setRenameCol(true)}>
            <LuPencil />
            重命名
          </MenuItem>
          <MenuItem className="text-txtprim text-sm gap-x-2" onClick={() => useStore.getState().addNewReqtoCol(col.id)}>
            <LuPlus />
            添加请求
          </MenuItem>
          <MenuItem className="text-txtprim text-sm gap-x-2" onClick={() => exportCollection()}>
            <LuDownload />
            导出
          </MenuItem>
          <MenuItem className="text-red-400 text-sm gap-x-2" onClick={() => onDeleteCol()}>
            <LuTrash />
            删除
          </MenuItem>
        </Menu>
      </div>
      <section {...getCollapseProps()}>
        {col.requests && col.requests.length ? (
          col.requests.map((a) => <RequestList req={a} key={a.id} />)
        ) : (
          <div className="pl-7">
            <p className="text-sm text-txtsec">未找到请求</p>
          </div>
        )}
      </section>
      {renameCol && <RenameCol renameCol={renameCol} setRenameCol={setRenameCol} col={col} />}
    </div>
  );
};

export default memo(Collection);
