import { toast } from "react-toastify";
import { useStore } from "../../store/store";
import CustomButton from "../misc/CustomButton";
import ModalLayout from "../misc/ModalLayout";

const RenameReq = ({ req, renameModal, setRenameModal }) => {
  let cLoading = useStore((x) => x.cLoading);
  const onRenameReq = async (e) => {
    e.preventDefault();
    let n = e.target.req_name.value;
    if (!n || n === "") {
      toast.warn("名称不能为空。");
      return;
    }
    let rsp = await useStore.getState().renameReq(req.coll_id, req.id, n);
    if (rsp) {
      setRenameModal(false);
      toast.success("请求重命名成功！");
    } else {
      toast.error("错误！无法重命名请求。");
    }
  };
  return (
    <ModalLayout open={renameModal} onClose={() => setRenameModal(false)} title="重命名请求">
      <form onSubmit={onRenameReq}>
        <div className="p-6">
          <p className="text-txtprim text-sm mb-2">请求名称</p>
          <input
            name="req_name"
            className="border border-txtsec text-lit w-full outline-none p-1 px-3 text-lg focus:border-txtprim rounded-sm"
            defaultValue={req?.name}
            required
            maxLength={100}
            autoFocus
            autoComplete="off"
          />
          <div className="w-full flex justify-end items-center mt-6 gap-x-4">
            <CustomButton name="重命名" type="submit" loading={cLoading} clx="px-4 py-1" />
            <CustomButton name="关闭" bg="bg-txtsec" clx="px-4 py-1" onClick={() => setRenameModal(false)} />
          </div>
        </div>
      </form>
    </ModalLayout>
  );
};

export default RenameReq;
