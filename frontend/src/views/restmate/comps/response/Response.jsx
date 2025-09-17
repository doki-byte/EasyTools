import React from "react";
import { Tab, TabList, TabPanel, Tabs } from "react-tabs";
import RspEditor from "./RspEditor";
import RspHeaders from "./RspHeaders";
import RspStatus from "./RspStatus";
import RspImage from "./RspImage";
import RspPdfContent from "./RspPdfContent";
import RspText from "./RspText";

const Response = ({ response }) => {
  const rspTypeRender = () => {
    switch (response?.contentType) {
      case "JSON":
      case "JAVASCRIPT":
      case "HTML":
        return <RspEditor lang={response.contentType} bodyContent={response?.bodyContent} />;
      case "IMAGE":
        return <RspImage bodyContent={response?.bodyContent} />;
      case "PDF":
        return <RspPdfContent bodyContent={response?.bodyContent} />;
      case "TEXT":
        return <RspText bodyContent={response?.bodyContent} />;
      default:
        return null;
    }
  };
  return (
    <Tabs style={{ height: "100%", width: "100%" }}>
      <div className="grid h-full w-full" style={{ gridTemplateRows: "24px minmax(0, 100%)", gridTemplateColumns: "minmax(0px, 100%)" }}>
        <div className="flex justify-between items-center">
          <TabList className="flex items-center h-full gap-x-4 text-sm">
            <Tab
              selectedClassName="!text-lit bg-brand !border-accent"
              className="inline-block outline-none h-full text-txtprim border-b-2 border-brand cursor-pointer"
            >
              Body
            </Tab>
            <Tab
              selectedClassName="!text-lit bg-brand !border-accent"
              className="inline-block outline-none h-full text-txtprim border-b-2 border-brand cursor-pointer"
            >
              Headers
            </Tab>
          </TabList>
          <RspStatus status={response?.statusCode} duration={response?.duration} httpStatus={response?.httpStatus} />
        </div>
        <div className="h-full w-full">
          <TabPanel style={{ height: "100%" }}>{rspTypeRender()}</TabPanel>
          <TabPanel style={{ height: "100%" }}>
            <RspHeaders headers={response?.headers} />
          </TabPanel>
        </div>
      </div>
    </Tabs>
  );
};

export default React.memo(Response);
