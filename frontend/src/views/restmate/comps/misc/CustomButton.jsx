import { LuLoaderCircle } from "react-icons/lu";

const CustomButton = ({ name, loading = false, bg = "bg-accent", color = "text-lit", clx, ...props }) => {
  return (
    <button
      className={`${bg} ${color} font-bold rounded-sm cursor-pointer flex justify-center items-center gap-x-2 ${loading ? "disabled:cursor-not-allowed" : `active:${bg}/80`} ${clx}`}
      disabled={loading}
      {...props}
    >
      {loading && <LuLoaderCircle className="animate-spin" />}
      {name}
    </button>
  );
};

export default CustomButton;
