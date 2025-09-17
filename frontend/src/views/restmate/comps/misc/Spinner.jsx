import { LuLoaderCircle } from "react-icons/lu";

const Spinner = ({ size = "32" }) => {
  return (
    <div className="text-accent">
      <LuLoaderCircle size={size} className="animate-spin" />
    </div>
  );
};

export default Spinner;
