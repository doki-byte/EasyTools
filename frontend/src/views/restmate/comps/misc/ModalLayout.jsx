import { LuX } from "react-icons/lu";

const ModalLayout = ({ children, onClose, title, open = false, size = "sm", header = true }) => {
  if (!open) return null;
  return (
    <div id="modal-layout" className="fixed inset-0" style={{ zIndex: 999 }}>
      <div className="flex justify-center items-center h-full w-full bg-black/50">
        <div className={`max-h-full bg-brand rounded-md ${size === "sm" ? "w-lg max-w-lg" : "w-2xl max-w-2xl"} border border-lines overflow-hidden`} onClick={(e) => e.stopPropagation()}>
          {header && (
            <div className="flex w-full justify-between text-lit gap-x-4 border-b border-lines pl-4 pr-2 pt-3 pb-2">
              <p className="font-bold text-lg">{title}</p>
              <div onClick={onClose} className="text-txtsec rounded-md hover:text-lit hover:bg-sec cursor-pointer p-1">
                <LuX size="22" />
              </div>
            </div>
          )}
          <div className="h-full">{children}</div>
        </div>
      </div>
    </div>
  );
};

export default ModalLayout;
