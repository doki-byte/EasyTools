import React from "react";

const RspPdfContent = ({ bodyContent }) => {
  return (
    <div className="pt-2 h-full w-full">
      <div className="h-full w-full border border-lines bg-lit">
        {bodyContent && <object data={bodyContent} type="application/pdf" width="100%" height="100%" />}
      </div>
    </div>
  );
};

export default React.memo(RspPdfContent);
