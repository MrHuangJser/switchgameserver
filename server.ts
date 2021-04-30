import {
  createReadStream,
  pathExists,
  pathExistsSync,
  readFile,
  readFileSync,
  stat,
  statSync,
  writeFile,
} from "fs-extra";
import { drive_v3, google } from "googleapis";
import { createServer } from "http";
import Path from "path";
import express from "express";

const app = express();
let inProcessing = false;
export interface Credentials {
  installed: Installed;
}

export interface Installed {
  client_id: string;
  project_id: string;
  auth_uri: string;
  token_uri: string;
  auth_provider_x509_cert_url: string;
  client_secret: string;
  redirect_uris: string[];
}
export interface GdriveToken {
  access_token: string;
  refresh_token: string;
}

app
  .use(async (req, res) => {
    try {
      if (inProcessing) {
        res.json({ files: [], success: "task is processing" });
      } else {
        if (!(await pathExists(Path.resolve(__dirname, "hbg.json")))) {
          res.status(200).json({ files: [], success: "no cache found,please request after few minutes" });
          inProcessing = true;
          await getFileList(await getGoogleAPI());
          inProcessing = false;
        } else {
          const indexStat = await stat(Path.resolve(__dirname, "hbg.json"));
          if (Date.now() - +indexStat.mtime > 1000 * 3600 * 24) {
            res.status(200).json({ files: [], success: "cache to old,please request after few minutes" });
            await getFileList(await getGoogleAPI());
          } else {
            res.sendFile(Path.resolve(__dirname, "hbg.json"), (err) => {
              if (err) {
                console.log(err);
              }
            });
          }
        }
      }
    } catch (e) {
      console.log(e);
    }
  })
  .listen(3001)
  .on("listening", () => {
    console.log("server listened on 3001");
  });

async function getGoogleAPI() {
  const {
    installed: { client_id, client_secret, redirect_uris },
  }: Credentials = JSON.parse((await readFile("credentials.json")).toString());
  const auth = new google.auth.OAuth2(client_id, client_secret, redirect_uris[0]);
  const token: GdriveToken = JSON.parse((await readFile("gdrive.token")).toString());
  auth.setCredentials(token);
  return google.drive({ version: "v3", auth });
}

async function getDriverList(api: drive_v3.Drive) {
  const drives = await api.drives.list({ pageSize: 90 });
  return drives.data.drives.filter((item) => /^hbg/.test(item.name));
}

function getNoCacheDriverList(drives: drive_v3.Schema$Drive[]) {
  return drives.filter((item) => {
    if (pathExistsSync(Path.resolve(__dirname, `${item.id}.json`))) {
      const stat = statSync(Path.resolve(__dirname, `${item.id}.json`));
      if (Date.now() - +stat.mtime > 1000 * 3600 * 24) {
        return true;
      }
    } else {
      return true;
    }
    return false;
  });
}

async function genHbgJSON(drives: drive_v3.Schema$Drive[]) {
  const allFiles = drives
    .reduce(
      (arr, item) => [
        ...arr,
        ...JSON.parse(readFileSync(Path.resolve(__dirname, `${item.id}.json`)).toString()),
      ],
      []
    )
    .map((item: drive_v3.Schema$File) => {
      const titleId = /(\[[0-9A-Fa-f]{16}\])/gi.exec(item.name);
      const resultVersionRegex = /(\[v[0-9]+\])/gi.exec(item.name);
      const version = resultVersionRegex ? resultVersionRegex[0] : "";
      return {
        size: +item.size,
        url: titleId
          ? `gdrive:${item.id}#${titleId[0]}${version}${Path.extname(item.name)}`
          : `gdrive:${item.name}`,
      };
    });
  return writeFile(
    Path.resolve(__dirname, "hbg.json"),
    JSON.stringify({ files: allFiles, success: "enjoy hbg shop!" }),
    { flag: "w" }
  );
}

async function getFileList(api: drive_v3.Drive) {
  const allDrives = await getDriverList(api);
  const drives = getNoCacheDriverList(allDrives);
  if (!drives.length) {
    return genHbgJSON(allDrives);
  }
  let list: drive_v3.Schema$File[] = [];
  return new Promise((resolve, reject) => {
    const recursive = (drive: drive_v3.Schema$Drive, pageToken?: any) => {
      api.files
        .list({
          fields: "nextPageToken, files(id, name, size)",
          q: `trashed = false and name contains '.nsz' or name contains '.nsp' or name contains '.xci'`,
          corpora: "drive",
          spaces: "drive",
          supportsAllDrives: true,
          includeItemsFromAllDrives: true,
          driveId: drive.id,
          pageSize: 1000,
          pageToken,
        })
        .then((res) => {
          list.push(...res.data.files);
          if (res.data.nextPageToken) {
            recursive(drive, res.data.nextPageToken);
          } else {
            writeFile(Path.resolve(__dirname, `${drive.id}.json`), JSON.stringify(list), {
              flag: "w",
            })
              .then(() => {
                list = [];
                if (drives.length) {
                  recursive(drives.shift());
                } else {
                  genHbgJSON(allDrives)
                    .then(() => resolve())
                    .catch((e) => reject(e));
                }
              })
              .catch((e) => reject(e));
          }
        })
        .catch((e) => reject(e));
    };
    recursive(drives.shift());
  });
}
