import { LuCookie } from "react-icons/lu";
import { useEffect, useState } from "react";
import { useStore } from "../../store/store";
import ModalLayout from "../misc/ModalLayout";
import CustomButton from "../misc/CustomButton";
import CookieDetail from "./CookieDetail";
import { ClearAllCookies, DeleteCookies, GetAllCookies } from "../../../../../wailsjs/go/restmate/RestMate";
import Spinner from "../misc/Spinner";

const CookieModal = () => {
  const [cookieLoad, setcookieLoad] = useState(true);
  const cookieModal = useStore((x) => x.cookieModal);
  const setCookieModal = useStore((x) => x.setCookieModal);
  const [cks, setCks] = useState(null);
  const getCooks = async () => {
    setcookieLoad(true);
    let r = await GetAllCookies();
    if (r.success && r.data && r.data.length) {
      let group = r?.data.reduce((acc, obj) => {
        const k = obj.Domain;
        if (!acc[k]) {
          acc[k] = [];
        }
        acc[k].push(obj);
        return acc;
      }, {});
      setCks(group);
      setcookieLoad(false);
    } else {
      setCks(null);
      setcookieLoad(false);
    }
  };
  useEffect(() => {
    getCooks();
  }, []);
  const deleteCookie = async (name) => {
    setcookieLoad(false);
    let r = await DeleteCookies(name);
    if (r.success && r.data && r.data.length) {
      let group = r?.data.reduce((acc, obj) => {
        const k = obj.Domain;
        if (!acc[k]) {
          acc[k] = [];
        }
        acc[k].push(obj);
        return acc;
      }, {});
      setCks(group);
      setcookieLoad(false);
    } else {
      setCks(null);
      setcookieLoad(false);
    }
  };
  const clearCookie = async () => {
    setcookieLoad(false);
    let r = await ClearAllCookies();
    if (r.success) {
      setCks(null);
    }
    setcookieLoad(false);
  };
  return (
    <ModalLayout open={cookieModal} onClose={() => setCookieModal(false)} title="Cookies" size="xl">
      <div className="h-full p-4 overflow-y-auto" style={{ maxHeight: "800px", height: "400px" }}>
        {cookieLoad ? (
          <div className="h-full w-full flex justify-center items-center">
            <Spinner />
          </div>
        ) : cks && typeof cks === "object" && Object.keys(cks).length ? (
          <div>
            <div className="flex items-center justify-end pb-2 pr-2">
              <div className="cursor-pointer" onClick={() => clearCookie()}>
                <p className="text-red-400 text-xs hover:underline">Clear all</p>
              </div>
            </div>
            {Object.keys(cks).map((x, id) => (
              <CookieDetail key={id} name={x} cookies={cks[x]} deleteCookie={deleteCookie} />
            ))}
          </div>
        ) : (
          <div className="h-full flex flex-col justify-center items-center py-6 gap-y-4 text-txtsec">
            <LuCookie size="96" />
            <div>
              <p className="text-sm mb-1 text-txtprim">Cookie Jar is empty.</p>
              <p className="text-sm text-txtprim">No Cookies available!</p>
            </div>
            <CustomButton name="Close" clx="px-4 py-1" onClick={() => setCookieModal(false)} />
          </div>
        )}
      </div>
    </ModalLayout>
  );
};

export default CookieModal;
