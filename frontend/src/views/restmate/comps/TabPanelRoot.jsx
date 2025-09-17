import { memo } from "react";
import ReqHead from "./ReqHead";
import ReqOptionTabs from "./reqOptions/ReqOptionTabs";
import Response from "./response/Response";
import BeforeResponse from "./response/BeforeResponse";
import { useStore } from "../store/store";
import { BarLoader } from "react-spinners";
import ErrorResponse from "./response/ErrorResponse";

const TabPanelRoot = ({ tab }) => {
  let invokeLoading = useStore((x) => x.invokeLoading);
  let sEnv = useStore((x) => x.envs.find((e) => e.selected));

  return (
    <div className="h-full grid pt-4" id="tabPanelRoot" style={{ gridTemplateRows: "min-content minmax(0,100%)", gridTemplateColumns: "minmax(0,100%)" }}>
      <ReqHead tabId={tab.id} method={tab.method} url={tab.url} name={tab.name} coll_id={tab.coll_id} envVars={sEnv?.variable} />
      <div
        className="h-full w-full grid py-4"
        style={{
          gridTemplateColumns: "minmax(0px, 100%) minmax(0px, 100%)",
          gridTemplateRows: "minmax(0, 100%)",
        }}
      >
        <div className="border-r border-lines px-6 h-full w-full">
          <ReqOptionTabs
            tabId={tab.id}
            reqTabInx={tab.reqTabInx || 0}
            params={tab.params}
            headers={tab.headers}
            bodyType={tab.body?.bodyType}
            bodyRaw={tab.body?.bodyRaw}
            formData={tab.body?.formData}
            envVars={sEnv?.variable}
          />
        </div>
        {/*no rsp and error handler here*/}

        <div className="h-full w-full px-6 relative">
          {invokeLoading && (
            <div className="absolute left-0 w-full px-2" style={{ top: "-4px" }}>
              <BarLoader width="100%" color="var(--color-accent)" height="1px" cssOverride={{ backgroundColor: "none" }} loading={invokeLoading} />
            </div>
          )}
          {!tab.response && <BeforeResponse />}
          {tab.response && (!tab.response.statusCode ? <ErrorResponse msg={tab.response.errorContent} /> : <Response response={tab.response} />)}
        </div>
      </div>
    </div>
  );
};

export default memo(TabPanelRoot);
