const ErrorResponse = ({ msg }) => {
  return (
    <div className="grid h-full w-full" style={{ gridTemplateRows: "min-content minmax(0, 100%)", gridTemplateColumns: "minmax(0px, 100%)" }}>
      <div className="h-full w-full">
        <p className="text-txtprim font-bold text-sm">响应</p>
      </div>
      <div className="pt-2 h-full px-6 flex justify-center items-center">
        <div className="bg-red-900/60 rounded-sm py-4 px-6" style={{ maxWidth: "600px" }}>
          <p className="text-txtprim text-sm italic line-clamp-6 break-all">{msg ? msg : "错误！请求失败。"}</p>
        </div>
      </div>
    </div>
  );
};

export default ErrorResponse;
