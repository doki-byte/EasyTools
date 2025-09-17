import { toast } from "react-toastify";
import { useStore } from "../../store/store";
import CustomButton from "../misc/CustomButton";
import ModalLayout from "../misc/ModalLayout";
import { useState } from "react";
import { LuInfo } from "react-icons/lu";

const MoveReq = ({ moveModal, setmoveModal, req_id, coll_id }) => {
  let cLoading = useStore((x) => x.cLoading);
  let cols = useStore((x) => x.collections);
  const [selcol, setSelcol] = useState(null);

  const onMoveReq = async () => {
    if (selcol === coll_id) return;
    if (!selcol || selcol === "") {
      toast.warn("请选择集合。");
      return;
    }
    let rsp = await useStore.getState().moveReq(req_id, coll_id, selcol);
    if (rsp) {
      setmoveModal(false);
      toast.success("请求移动成功！");
    } else {
      toast.error("错误！无法移动请求。");
    }
  };
  return (
    <ModalLayout open={moveModal} onClose={() => setmoveModal(false)} title="Move Request">
      <div className="p-6">
        <div className="">
          <p className="text-txtprim text-sm">Select Collection</p>
        </div>
        {cols && cols.length && cols.filter((x) => x.id !== coll_id).length ? (
          <div className="bg-sec mt-2 border border-lines overflow-y-auto" style={{ maxHeight: "300px" }}>
            {cols.map((x) =>
              x.id === coll_id ? null : (
                <div
                  className={`${x.id === selcol ? "bg-txtprim text-brand" : "bg-sec text-txtprim"} rounded-sm p-2 cursor-pointer`}
                  key={x.id}
                  onClick={() => setSelcol(x.id)}
                >
                  <p className="text-sm truncate whitespace-nowrap overflow-ellipsis" style={{ maxWidth: "90%" }}>
                    {x.name}
                  </p>
                </div>
              ),
            )}
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
          <CustomButton name="移动" type="submit" loading={cLoading} clx="px-4 py-1" onClick={onMoveReq} />
          <CustomButton name="关闭" bg="bg-txtsec" clx="px-4 py-1" onClick={() => setmoveModal(false)} />
        </div>
      </div>
    </ModalLayout>
  );
};

export default MoveReq;
