import React from "react";
import { Tab, TabList, TabPanel, Tabs } from "react-tabs";
import ReqParams from "./ReqParams";
import ReqHeaders from "./ReqHeaders";
import ReqBodyOption from "./ReqBodyOption";
import BodyFormData from "./BodyFormData";
import BodyJson from "./BodyJson";
import { useStore } from "../../store/store";

const ReqOptionTabs = ({ tabId, params, headers, bodyType, bodyRaw, formData, envVars, reqTabInx }) => {
  return (
    <div className="h-full w-full">
      <Tabs style={{ height: "100%", width: "100%" }} selectedIndex={reqTabInx} onSelect={(i) => useStore.getState().setReqTabInx(tabId, i)}>
        <div className="grid h-full w-full" style={{ gridTemplateRows: "24px minmax(0, 100%)", gridTemplateColumns: "minmax(0px, 100%)" }}>
          <div className="flex justify-between items-center">
            <TabList className="flex items-center h-full gap-x-4 text-sm">
              <Tab
                selectedClassName="!text-lit bg-brand !border-accent"
                className="inline-block outline-none h-full text-txtprim border-b-2 border-brand cursor-pointer"
              >
                Params
              </Tab>
              <Tab
                selectedClassName="!text-lit bg-brand !border-accent"
                className="inline-block outline-none h-full text-txtprim border-b-2 border-brand cursor-pointer"
              >
                Headers
              </Tab>
              <Tab
                selectedClassName="!text-lit bg-brand !border-accent"
                className="inline-block outline-none h-full text-txtprim border-b-2 border-brand cursor-pointer"
              >
                Body
              </Tab>
            </TabList>
            <div className="cursor-pointer" onClick={() => useStore.getState().setCookieModal(true)}>
              <p className="text-xs bg-brand text-blue-400 font-bold">Cookies</p>
            </div>
          </div>
          <div className="h-full w-full">
            <TabPanel style={{ height: "100%" }}>
              <ReqParams params={params} tabId={tabId} envVars={envVars} />
            </TabPanel>
            <TabPanel style={{ height: "100%" }}>
              <ReqHeaders headers={headers} tabId={tabId} envVars={envVars} />
            </TabPanel>
            <TabPanel style={{ height: "100%", width: "100%" }}>
              <div className="pt-2 h-full grid w-full" style={{ gridTemplateRows: "min-content minmax(0,100%)", gridTemplateColumns: "minmax(0px, 100%)" }}>
                <ReqBodyOption tabId={tabId} bodyType={bodyType} />
                {bodyType === "json" ? (
                  <BodyJson tabId={tabId} bodyRaw={bodyRaw} envVars={envVars} />
                ) : bodyType === "formdata" ? (
                  <BodyFormData tabId={tabId} formData={formData} envVars={envVars} />
                ) : null}
              </div>
            </TabPanel>
          </div>
        </div>
      </Tabs>
    </div>
  );
};

export default React.memo(ReqOptionTabs);
