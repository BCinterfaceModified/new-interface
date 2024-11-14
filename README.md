# New-Interface

기존 인터페이스 서버에서 필수적인 부분만 추출하고, commu-module과 상호작용할 수 있게 수정함.

---

### How to use

Mongo replica set을 사용하기 위해서 key 파일을 생성해야한다.

```bash
~$ sudo ./mongoKeySetup.sh
```

실행하면 rs_keyfile이 생성된다.

다음으로 해당 interface server를 docker에서 작동하기 위해 아래 명령어를 실행해 도커 이미지파일을 만들어준다.

```bash
~$ ./interfaceDockerBuild.sh 0.1
```

0.1은 버전이다. 해당 값은 임의대로 입력해도 상관없으나 이 경우, `docker-compose.yml`파일의  `bc1`의`image`에 bcinterface:0.1의 버전값을 임의대로 입력한것과 동일하게 변경해주어야한다.

이제 만들어진 이미지 파일로 인터페이스 서버를 실행해보자.

```bash
~$ docker compose up -d
```

성공적으로 컨테이너가 생성되었다면, 마지막으로 3개의 mongoDB를 하나로 묶어주어 클러스터링 모드로 작동할 수 있게 설정해주어야 한다.

```bash
#해당 명령어로 mongo1 container 명령줄에 접근
~$ docker exec -it mongo1 /bin/bash

#해당 명령어로 mongo shell 접근
mongo -u root -p root

#해당 명령어를 입력해 mongo1을 mongo2, 3와 replica set으로 묶음
>  rs.initiate(
  {
    _id: "rs0",
    version: 1,
    members: [
      { _id: 0, host: "mongo1:27017" },
      { _id: 1, host: "mongo2:27018" },
      { _id: 2, host: "mongo3:27019" }
    ]
  }
)
```

위의 명령어 순서를 따라 mongo replica set을 만들어주었다면 아래 명령어로 상태를 확인하는 것이 가능하다.

```bash
> rs.status()
```

---

### Extra

mongoDB 조회시에 `rs0:SECONDARY>`로 명령줄이 표시되고, secondary에 대해 데이터 조회를 할 수 없다는 `"errmsg" : "not master and slaveOk=false"`오류메시지 출력 시

```bash
> rs.secondaryOk()
```

해당 명령어를 통해 Secondary 노드에서도 데이터 조회가 가능하다

**mongoDB 기본적인 명령어**

```bash
#모든 DB조회
> show dbs

#특정 DB를 사용
> use <DBname>

#특정 DB에 use된 상태에서 모든 collection 조회
> show collections

#특정 collection 내부 데이터를 모두 조회하고 싶을 때
> db.<collection>.find().pretty()

#특정 collection 내부 모든 데이터를 삭제(Primary노드에서만 가능)
> db.<collection>.deletemany({})

#mongoDB 접속 시에 port 지정
> mongo -port 27018 -u root -p root
```