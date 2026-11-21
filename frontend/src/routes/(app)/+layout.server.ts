   import { redirect } from "@sveltejs/kit";                                                                                                                                                   
   import type { LayoutServerLoad } from "./$types";                                                                                                                                           
                                                                                                                                                                                               
   export const load: LayoutServerLoad = async ({ parent }) => {                                                                                                                               
     const { isAuthenticated } = await parent();                                                                                                                                               
     if (!isAuthenticated) redirect(303, "/auth/login");                                                                                                                                       
   };    
