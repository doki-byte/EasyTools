import { toast } from "react-toastify";
import { useStore } from "../../store/store";
import CustomButton from "../misc/CustomButton";
import ModalLayout from "../misc/ModalLayout";

const RenameCol = ({ col, renameCol, setRenameCol }) => {
  let cLoading = useStore((x) => x.cLoading);
  const onRenameCol = async (e) => {
    e.preventDefault();
    let n = e.target.col_name.value;
    if (!n || n === "") {
      toast.warn("名称不能为空。");
      return;
    }
    let rsp = await useStore.getState().renameCollection(col.id, n);
    if (rsp) {
      setRenameCol(false);
      toast.success("集合重命名成功！");
    } else {
      toast.error("错误！无法重命名集合。");
    }
  };
  return (
    <ModalLayout open={renameCol} onClose={() => setRenameCol(false)} title="重命名集合">
      <form onSubmit={onRenameCol}>
        <div className="p-6">
          <p className="text-txtprim text-sm mb-2">集合名称</p>
          <input
            name="col_name"
            className="border border-txtsec text-lit w-full outline-none p-1 px-3 text-lg focus:border-txtprim rounded-sm"
            defaultValue={col?.name}
            required
            maxLength={100}
            autoFocus
            autoComplete="off"
          />
          <div className="w-full flex justify-end items-center mt-6 gap-x-4">
            <CustomButton name="重命名" type="submit" loading={cLoading} clx="px-4 py-1" />
            <CustomButton name="关闭" bg="bg-txtsec" clx="px-4 py-1" onClick={() => setRenameCol(false)} />
          </div>
        </div>
      </form>
    </ModalLayout>
  );
};

export default RenameCol;
