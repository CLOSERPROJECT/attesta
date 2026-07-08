import postcssCustomMedia from "postcss-custom-media";

export default {
  plugins: [
    postcssCustomMedia({ preserve: false }),
  ],
};
