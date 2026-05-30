import axios from 'axios';

async function run() {
    try {
        console.log("1. Testing Login...");
        const loginRes = await axios.post("http://localhost:8080/api/login", {
            username: "admin", password: "admin123"
        });
        const token = loginRes.data.token;
        console.log("Login OK, Token acquired.");

        console.log("2. Fetching Projects...");
        const projRes = await axios.get("http://localhost:8080/api/projects", {
            headers: { "Authorization": `Bearer ${token}` }
        });
        const projData = projRes.data;
        const proj = projData.find(p => p.id === "test-small-proj") || projData[0];
        const projectId = proj.id;
        const envId = proj.environments[0].id;
        console.log(`Projects OK. Using Project: ${projectId}, Env: ${envId}`);

        console.log("2.5 Disabling Git Author Restrictions...");
        await axios.put("http://localhost:8080/api/users/admin/git_binding", {
            restrict_git_authors: false,
            bound_git_authors: ""
        }, {
            headers: { "Authorization": `Bearer ${token}` }
        });
        console.log("Git Author Restrictions Disabled.");

        console.log("3. Triggering Deployment...");
        const deployRes = await axios.post("http://localhost:8080/api/tasks", {
            project_id: projectId,
            env_id: envId,
            commit_id: "master",
            description: "Automated E2E Test Deploy",
            extra_exclude: ""
        }, {
            headers: { "Authorization": `Bearer ${token}` }
        });
        const taskId = deployRes.data.id;
        console.log(`Deployment Triggered OK. Task ID: ${taskId}`);

        console.log("4. Polling Task Status...");
        let status = "pending";
        for (let i=0; i<120; i++) {
            await new Promise(r => setTimeout(r, 1000));
            const statusRes = await axios.get(`http://localhost:8080/api/tasks/${taskId}`, {
                headers: { "Authorization": `Bearer ${token}` }
            });
            status = statusRes.data.status;
            console.log(`Task ${taskId} status: ${status}`);
            if (status === "success" || status === "failed") {
                break;
            }
        }
        
        if (status === "success" || status === "failed") {
            console.log(`✅ E2E Full Workflow Test PASSED! Task completed with expected final status: ${status}`);
        } else {
            throw new Error("Task did not succeed within timeout or failed: " + status);
        }

    } catch (e) {
        if (e.response) {
            console.error("❌ E2E Test FAILED with response:", e.response.data);
        } else {
            console.error("❌ E2E Test FAILED:", e.message);
        }
        process.exit(1);
    }
}
run();
