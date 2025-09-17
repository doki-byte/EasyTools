import { Bounce, ToastContainer } from "react-toastify";
import SideBar from "./SideBar";
import { useStore } from "../../store/store";
import CookieModal from "../cookies/CookieModal";

const Layout = ({ children }) => {
  const cookieModal = useStore((x) => x.cookieModal);
  return (
    <div className="h-svh bg-brand max-h-svh relative overflow-hidden" id="layout">
      <div className="grid h-full" style={{ gridTemplateColumns: "auto 1fr", gridTemplateRows: "minmax(0, 100%)" }}>
        <div id="sidenav">
          <SideBar />
        </div>
        <div className="w-full h-full" style={{ minWidth: 0 }}>
          {children}
        </div>
      </div>
      {cookieModal && <CookieModal />}
      <ToastContainer
        position="bottom-center"
        autoClose={3000}
        hideProgressBar={true}
        newestOnTop={false}
        closeOnClick
        rtl={false}
        pauseOnFocusLoss={false}
        draggable
        pauseOnHover
        theme="light"
        transition={Bounce}
      />
    </div>
  );
};

export default Layout;
