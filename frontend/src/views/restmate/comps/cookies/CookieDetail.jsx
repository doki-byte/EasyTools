import { memo } from "react";
import { useCollapse } from "react-collapsed";
import { LuChevronDown, LuChevronRight, LuX } from "react-icons/lu";

const CookieDetail = ({ name, cookies, deleteCookie }) => {
  const { getCollapseProps, getToggleProps, isExpanded } = useCollapse();
  function formatCookie(cookieObj) {
    if (!cookieObj.Name) {
      return "";
    }
    let cookieStr = `${cookieObj.Name ? cookieObj.Name : ""}=${cookieObj.Value ? cookieObj.Value : ""}; `;
    if (cookieObj.Path) {
      cookieStr += `Path=${cookieObj.Path}; `;
    }
    if (cookieObj.Domain) {
      cookieStr += `Domain=${cookieObj.Domain}; `;
    }
    if (cookieObj.Secure) {
      cookieStr += "Secure; ";
    }
    if (cookieObj.HttpOnly) {
      cookieStr += "HttpOnly; ";
    }
    if (cookieObj.Expires) {
      const expires = new Date(cookieObj.Expires).toUTCString();
      cookieStr += `Expires=${expires}; `;
    }
    return cookieStr.trim();
  }
  return (
    <div className="mb-4">
      <div className="flex justify-between items-center bg-sec p-2 rounded-md">
        <div {...getToggleProps()} className="text-txtprim grow overflow-hidden cursor-pointer flex items-center justify-start gap-x-1">
          <div>{isExpanded ? <LuChevronDown size="14" /> : <LuChevronRight size="14" />}</div>
          <p className="truncate whitespace-nowrap overflow-ellipsis text-sm" style={{ width: "90%" }}>
            {name ? name : ""}
          </p>
        </div>
        <div className="text-txtprim cursor-pointer hover:text-red-400" onClick={() => deleteCookie(name)}>
          <LuX />
        </div>
      </div>
      {cookies && cookies.length ? (
        <section {...getCollapseProps()}>
          {cookies.map((c, id) => (
            <div key={id} className="text-txtprim bg-gray-600/40 p-2 w-full rounded-md mt-2">
              <p className="text-xs break-all break-words">{formatCookie(c)}</p>
            </div>
          ))}
        </section>
      ) : null}
    </div>
  );
};

export default memo(CookieDetail);
