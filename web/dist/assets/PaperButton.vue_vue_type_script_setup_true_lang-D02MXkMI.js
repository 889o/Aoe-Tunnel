import{c as l,b as r}from"./utils-BTwVWYNe.js";import{d,c as p,B as c,e as a,x as i,g as f,af as u,k as t}from"./index-D6abuIX6.js";/**
 * @license lucide-vue-next v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const m=l("LoaderCircleIcon",[["path",{d:"M21 12a9 9 0 1 1-6.219-8.56",key:"13zald"}]]),b=["disabled"],B=d({__name:"PaperButton",props:{variant:{default:"default"},size:{default:"md"},loading:{type:Boolean,default:!1},disabled:{type:Boolean,default:!1},block:{type:Boolean,default:!1}},setup(s){const e=s,o={default:"",primary:"paper-btn-primary",ghost:"paper-btn-ghost",danger:"paper-btn-danger"};return(n,g)=>(t(),p("button",{type:"button",disabled:e.disabled||e.loading,class:c(a(r)("paper-btn",o[e.variant],e.size==="sm"&&"paper-btn-sm",e.block&&"w-full"))},[e.loading?(t(),i(a(m),{key:0,class:"size-4 animate-spin"})):f("",!0),u(n.$slots,"default")],10,b))}});export{B as _};
