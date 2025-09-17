import React from "react";

const RspImage = ({ bodyContent }) => {
  return (
    <div className="pt-2 h-full w-full">
      <div className="h-full w-full border border-lines bg-lit">
        <div className="flex justify-center items-center w-full h-full">
          {bodyContent && <img src={bodyContent} className="max-h-full max-w-full object-center object-contain" />}
        </div>
      </div>
    </div>
  );
};

export default React.memo(RspImage);
